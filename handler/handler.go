package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/andres-movl/gojsonschema"
	"github.com/gin-gonic/gin"
	"movinglake.com/haven/handler/jsonutils"
	"movinglake.com/haven/wrappers"
)

// Has the actual logic of the API in easy to test functions.

// We need to hold DB connections.
type HavenHandler struct {
	db wrappers.DB
}

func NewHavenHandler(db wrappers.DB) *HavenHandler {
	return &HavenHandler{
		db: db,
	}
}

type APIResponse struct {
	Error string `json:"error"`
}

type AddPayloadRequest struct {
	Resource string      `json:"resource"`
	Payload  interface{} `json:"payload"`
}

type AddPayloadResponse struct {
	APIResponse
	Success bool `json:"success"`
}

type ValidatePayloadRequest struct {
	Resource string      `json:"resource"`
	Payload  interface{} `json:"payload"`
}

type ValidatePayloadResponse struct {
	APIResponse
	Valid            bool            `json:"valid"`
	ValidationErrors []ErrorResponse `json:"validation_errors"`
}

type GetSchemaResponse struct {
	APIResponse
	Schema map[string]any `json:"schema"`
}

type SetSchemaRequest struct {
	Resource string         `json:"resource"`
	Schema   map[string]any `json:"schema"`
}

type SetSchemaResponse struct {
	APIResponse
	Success bool `json:"success"`
}

type GetAllResourcesResponse struct {
	APIResponse
	Resources []wrappers.Resource `json:"resources"`
}

type GetResourceVersionsResponse struct {
	APIResponse
	Versions []wrappers.ResourceVersions `json:"versions"`
}

type GetReferencePayloadResponse struct {
	APIResponse
	Payload interface{} `json:"payload"`
}

// addPayload adds a new payload to the specific resource.
func (h *HavenHandler) addPayload(c *gin.Context) {
	var request AddPayloadRequest
	var response AddPayloadResponse
	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		response.Error = err.Error()
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get the schema of the resource.
	t := h.db.OpenTxn()
	r := &wrappers.Resource{}
	t.Find(r, "name = ?", request.Resource)

	schema := make(map[string]interface{})
	if r.ID != 0 {
		if err := json.Unmarshal([]byte(r.Schema), &schema); err != nil {
			response.Error = fmt.Sprintf("failed to unmarshal schema: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}
	}

	newSchema, err := jsonutils.ApplyPayload(schema, request.Payload, request.Resource)
	if err != nil {
		response.Error = fmt.Sprintf("failed to apply payload: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	if newSchema == nil {
		// No changes to existing schema.
		log.Printf("no changes to the schema for resource %v", r)
		response.Success = true
		c.JSON(http.StatusOK, response)
		return
	}
	log.Printf("changes found to the schema for resource %s", request.Resource)

	// Save the new schema.
	newSchemaBytes, err := json.Marshal(newSchema)
	if err != nil {
		response.Error = fmt.Sprintf("failed to marshal new schema: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	r.Version += 1
	oldSchema := r.Schema
	r.Schema = string(newSchemaBytes)
	r.Name = request.Resource
	t.Save(r)

	// Save the reference payload.
	payloadBytes, err := json.Marshal(request.Payload)
	if err != nil {
		response.Error = fmt.Sprintf("failed to marshal payload: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	refPayload := &wrappers.ReferencePayloads{
		ResourceID: r.ID,
		Payload:    string(payloadBytes),
	}
	t.Save(refPayload)

	// Save the new version.
	rv := &wrappers.ResourceVersions{
		ResourceID:          r.ID,
		ReferencePayloadsID: refPayload.ID,
		OldSchema:           oldSchema,
		NewSchema:           string(newSchemaBytes),
		Version:             r.Version,
	}

	t.Save(rv)
	t.Commit()
	response.Success = true
	c.JSON(http.StatusOK, response)
}

type ErrorResponse struct {
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Context     map[string]any `json:"context"`
}

func toPath(ctx *gojsonschema.JsonContext) string {
	var path string
	for ctx != nil {
		path = ctx.Head() + "." + path
		ctx = ctx.Tail()
	}
	return path
}

// validatePayload validates the payload against the schema.
func (h *HavenHandler) validatePayload(c *gin.Context) {
	var request ValidatePayloadRequest
	var response ValidatePayloadResponse
	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		response.Error = err.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	// Get the schema of the resource.
	res, err := h.db.GetResource(request.Resource)
	if err != nil {
		response.Error = fmt.Sprintf("failed to get resource from db: %v", err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	schema := make(map[string]any)
	if err := json.Unmarshal([]byte(res.Schema), &schema); err != nil {
		response.Error = fmt.Sprintf("failed to unmarshal schema: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	result, err := jsonutils.ValidatePayload(schema, request.Payload)
	if err != nil {
		response.Error = fmt.Sprintf("failed to validate payload: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	if !result.Valid() {
		var errs []ErrorResponse
		for _, e := range result.Errors() {
			errs = append(errs, ErrorResponse{
				Type:        e.Type(),
				Description: e.Description(),
				Context: map[string]any{
					"field":    e.Details()["field"],
					"property": e.Details()["property"],
					"expected": e.Details()["expected"],
					"given":    e.Details()["given"],
					"path":     toPath(e.Context()),
				},
			})
		}
		response.Valid = false
		response.ValidationErrors = errs
		c.JSON(http.StatusOK, response)
		return
	}

	response.Valid = true
	c.JSON(http.StatusOK, response)
}

// getSchema returns the schema of the resource.
func (h *HavenHandler) getSchema(c *gin.Context) {
	var response GetSchemaResponse
	res, err := h.db.GetResource(c.Params.ByName("name"))
	if err != nil {
		response.Error = fmt.Sprintf("failed to get resource from db: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response.Schema = make(map[string]any)
	if err := json.Unmarshal([]byte(res.Schema), &response.Schema); err != nil {
		response.Error = fmt.Sprintf("failed to unmarshal DB schema: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	c.JSON(http.StatusOK, response)
}

// setSchema sets the schema of the resource.
func (h *HavenHandler) setSchema(c *gin.Context) {
	var request SetSchemaRequest
	var response SetSchemaResponse
	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		response.Error = fmt.Sprintf("failed to parse json request: %v", err)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	res, err := h.db.GetResource(request.Resource)
	if err != nil {
		m, err := json.Marshal(request.Schema)
		if err != nil {
			response.Error = fmt.Sprintf("failed to marshal schema: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		// Create a new resource.
		res = &wrappers.Resource{
			Name:    request.Resource,
			Schema:  string(m),
			Version: 1,
		}
		t := h.db.OpenTxn()
		t.Save(res)

		// Save the new version.
		rv := &wrappers.ResourceVersions{
			ResourceID:          res.ID,
			ReferencePayloadsID: 0,
			OldSchema:           "",
			NewSchema:           res.Schema,
			Version:             res.Version,
		}
		t.Save(rv)
		t.Commit()
		response.Success = true
		c.JSON(http.StatusOK, response)
		return
	}
	schemaBytes, err := json.Marshal(request.Schema)
	if err != nil {
		response.Error = fmt.Sprintf("failed to marshal schema: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	oldSchema := res.Schema
	res.Schema = string(schemaBytes)
	res.Version += 1
	t := h.db.OpenTxn()
	t.Save(res)

	// Save the new version.
	rv := &wrappers.ResourceVersions{
		ResourceID:          res.ID,
		ReferencePayloadsID: 0,
		OldSchema:           oldSchema,
		NewSchema:           res.Schema,
		Version:             res.Version,
	}
	t.Save(rv)
	t.Commit()
	response.Success = true
	c.JSON(http.StatusOK, response)
}

// getResources returns all the resources.
func (h *HavenHandler) getResources(c *gin.Context) {
	var response GetAllResourcesResponse
	resources, err := h.db.GetAllResources()
	if err != nil {
		response.Error = fmt.Sprintf("failed to get resources from db: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response.Resources = resources
	c.JSON(http.StatusOK, response)
}

// getResourceVersions returns all the versions of the resource.
func (h *HavenHandler) getResourceVersions(c *gin.Context) {
	var response GetResourceVersionsResponse
	idStr := c.Params.ByName("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error = fmt.Sprintf("failed to parse id: %v", err)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	versions, err := h.db.GetResourceVersions(uint(id))
	if err != nil {
		response.Error = fmt.Sprintf("failed to get resource versions from db: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response.Versions = versions
	c.JSON(http.StatusOK, response)
}

// getReferencePayload returns the reference payload of the version.
func (h *HavenHandler) getReferencePayload(c *gin.Context) {
	var response GetReferencePayloadResponse
	idStr := c.Params.ByName("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error = fmt.Sprintf("failed to parse id: %v", err)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	payload, err := h.db.GetReferencePayload(uint(id))
	if err != nil {
		response.Error = fmt.Sprintf("failed to get reference payload from db: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response.Payload = payload
	c.JSON(http.StatusOK, response)
}

func (h *HavenHandler) RegisterRoutes(e *gin.Engine) error {
	e.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})
	e.POST("/api/v1/add_payload", h.addPayload)
	e.POST("/api/v1/validate_payload", h.validatePayload)
	e.GET("/api/v1/get_schema/:name", h.getSchema)
	e.POST("/api/v1/set_schema", h.setSchema)
	e.GET("/api/v1/get_all_resources", h.getResources)
	e.GET("/api/v1/get_resource_versions/:id", h.getResourceVersions)
	e.GET("/api/v1/get_reference_payload/:id", h.getReferencePayload)
	return nil
}
