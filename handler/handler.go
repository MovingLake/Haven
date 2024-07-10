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
	"movinglake.com/haven/handler/notifications"
	"movinglake.com/haven/wrappers"
)

// Has the actual logic of the API in easy to test functions.

// We need to hold DB connections.
type HavenHandler struct {
	db      wrappers.DB
	slacker notifications.Sender
}

// NotificationsConfig holds the configuration for notifications.
type NotificationsConfig struct {
	SlackToken     string
	SlackChannelID string
}

func NewHavenHandler(db wrappers.DB, nc *NotificationsConfig) *HavenHandler {
	handler := &HavenHandler{
		db: db,
	}
	if nc != nil {
		handler.slacker = notifications.NewSlackSender(nc.SlackToken, nc.SlackChannelID)
	}
	return handler
}

type APIResponse struct {
	Error string `json:"error"`
}

type AddPayloadRequest struct {
	Resource string      `json:"resource"`
	Payload  interface{} `json:"payload"`
}

type ResourceResp struct {
	ID      uint           `json:"id"`
	Name    string         `json:"name"`
	Schema  map[string]any `json:"schema"`
	Version uint           `json:"version"`
}

type AddPayloadResponse struct {
	APIResponse
	Success  bool         `json:"success"`
	Resource ResourceResp `json:"resource"`
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
	Resource ResourceResp `json:"resource"`
	Success  bool         `json:"success"`
}

type GetAllResourcesResponse struct {
	APIResponse
	Resources []ResourceResp `json:"resources"`
}

type ResourceVersionsResponse struct {
	ID                  uint           `json:"id"`
	Version             uint           `json:"version"`
	ResourceID          uint           `json:"resource_id"`
	ReferencePayloadsID uint           `json:"reference_payloads_id"`
	OldSchema           map[string]any `json:"old_schema"`
	NewSchema           map[string]any `json:"new_schema"`
}

type GetResourceVersionsResponse struct {
	APIResponse
	Versions []ResourceVersionsResponse `json:"versions"`
}

type GetReferencePayloadResponse struct {
	APIResponse
	ID      uint        `json:"id"`
	Payload interface{} `json:"payload"`
}

// addPayload adds a new payload to the specific resource.
func (h *HavenHandler) addPayload(c *gin.Context) {
	var request AddPayloadRequest
	var response AddPayloadResponse
	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		response.Error = fmt.Sprintf("failed to parse json request: %v", err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get the schema of the resource.
	t := h.db.OpenTxn()
	r := &wrappers.Resource{}
	if err := h.db.Find(r, t, "name = ?", request.Resource); err != nil {
		response.Error = fmt.Sprintf("failed to get resource from db: %v", err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

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
		log.Printf("no changes to the schema for resource %v", request.Resource)
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
	if err := h.db.Save(r, t); err != nil {
		response.Error = fmt.Sprintf("failed to save new schema: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

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
	if err := h.db.Save(refPayload, t); err != nil {
		response.Error = fmt.Sprintf("failed to save reference payload: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Save the new version.
	rv := &wrappers.ResourceVersions{
		ResourceID:          r.ID,
		ReferencePayloadsID: refPayload.ID,
		OldSchema:           oldSchema,
		NewSchema:           string(newSchemaBytes),
		Version:             r.Version,
	}

	if err := h.db.Save(rv, t); err != nil {
		response.Error = fmt.Sprintf("failed to save new version: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	if err := h.db.Commit(t); err != nil {
		response.Error = fmt.Sprintf("failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	if h.slacker != nil && h.slacker.IsActive() {
		log.Printf("sending slack message for new version of schema for resource %s", request.Resource)
		err := h.slacker.SendMessage(
			fmt.Sprintf("New version `%d` of schema for resource `%s` has been added",
				r.Version,
				request.Resource))
		if err != nil {
			log.Printf("failed to send slack message: %v", err)
		}
	} else {
		log.Printf("slack not configured, skipping sending message for new version of schema for resource %s", request.Resource)
	}
	response.Success = true
	var schemaMap map[string]any
	if err := json.Unmarshal(newSchemaBytes, &schemaMap); err != nil {
		response.Error = fmt.Sprintf("failed to unmarshal new schema: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response.Resource = ResourceResp{
		ID:      r.ID,
		Name:    r.Name,
		Schema:  schemaMap,
		Version: r.Version,
	}
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
		if err := h.db.Save(res, t); err != nil {
			response.Error = fmt.Sprintf("failed to save resource: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}

		// Save the new version.
		rv := &wrappers.ResourceVersions{
			ResourceID:          res.ID,
			ReferencePayloadsID: 0,
			OldSchema:           "",
			NewSchema:           res.Schema,
			Version:             res.Version,
		}
		if err := h.db.Save(rv, t); err != nil {
			response.Error = fmt.Sprintf("failed to save resource version: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}

		if err := h.db.Commit(t); err != nil {
			response.Error = fmt.Sprintf("failed to commit transaction: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}
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
	if err := h.db.Save(res, t); err != nil {
		response.Error = fmt.Sprintf("failed to save resource: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Save the new version.
	rv := &wrappers.ResourceVersions{
		ResourceID:          res.ID,
		ReferencePayloadsID: 0,
		OldSchema:           oldSchema,
		NewSchema:           res.Schema,
		Version:             res.Version,
	}
	if err := h.db.Save(rv, t); err != nil {
		response.Error = fmt.Sprintf("failed to save resource version: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	if err := h.db.Commit(t); err != nil {
		response.Error = fmt.Sprintf("failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response.Success = true
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		response.Error = fmt.Sprintf("failed to unmarshal schema: %v", err)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response.Resource = ResourceResp{
		ID:      res.ID,
		Name:    res.Name,
		Schema:  schema,
		Version: res.Version,
	}
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
	for _, r := range resources {
		var schema map[string]any
		if err := json.Unmarshal([]byte(r.Schema), &schema); err != nil {
			response.Error = fmt.Sprintf("failed to unmarshal schema: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		response.Resources = append(response.Resources, ResourceResp{
			Name:    r.Name,
			Schema:  schema,
			Version: r.Version,
		})
	}
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
	if len(versions) == 0 {
		response.Error = fmt.Sprintf("no versions found for resource id %d", id)
		c.JSON(http.StatusNotFound, response)
		return
	}
	for _, v := range versions {
		var oldSchema map[string]any
		if v.OldSchema == "" {
			oldSchema = make(map[string]any)
		} else if err := json.Unmarshal([]byte(v.OldSchema), &oldSchema); err != nil {
			response.Error = fmt.Sprintf("failed to unmarshal old schema: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		var newSchema map[string]any
		if err := json.Unmarshal([]byte(v.NewSchema), &newSchema); err != nil {
			response.Error = fmt.Sprintf("failed to unmarshal new schema: %v", err)
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		response.Versions = append(response.Versions, ResourceVersionsResponse{
			Version:             v.Version,
			ResourceID:          v.ResourceID,
			ReferencePayloadsID: v.ReferencePayloadsID,
			OldSchema:           oldSchema,
			NewSchema:           newSchema,
		})
	}
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
	if payload == nil {
		response.Error = fmt.Sprintf("no reference payload found for id %d", id)
		c.JSON(http.StatusNotFound, response)
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
