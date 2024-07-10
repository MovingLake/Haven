package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"movinglake.com/haven/wrappers"
)

func TestAddPayload(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	// Create a new HavenHandler with the fake DB
	handler := NewHavenHandler(db, nil)

	// Create a test router
	router := gin.Default()
	router.POST("/add-payload", handler.addPayload)

	// Create a test request
	payload := AddPayloadRequest{
		Resource: "users",
		Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
	}
	out, _ := json.Marshal(payload)
	request := httptest.NewRequest(http.MethodPost, "/add-payload", bytes.NewBuffer(out))

	// Perform the request
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// Assert the response
	assert.Equal(t, http.StatusOK, response.Code)
	resp := &AddPayloadResponse{}
	json.Unmarshal(response.Body.Bytes(), resp)
	assert.Equal(t, payload.Resource, resp.Resource.Name)
	//		"{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",

	assert.Equal(t,
		map[string]any{
			"$id":                  "https://movinglake.com/haven.schema.json",
			"$schema":              "https://json-schema.org/draft/2020-12/schema",
			"additionalProperties": false,
			"properties": map[string]any{
				"age":  map[string]any{"type": "number"},
				"name": map[string]any{"type": "string"},
			},
			"required": []any{"age", "name"},
			"title":    "users",
			"type":     "object",
		},
		resp.Resource.Schema)
}

func TestValidatePayload(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	db.Save(&wrappers.Resource{
		Model:   gorm.Model{ID: 1},
		Name:    "users",
		Schema:  "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
		Version: 1,
	}, nil)

	// Create a new HavenHandler with the fake DB
	handler := NewHavenHandler(db, nil)

	// Create a test router
	router := gin.Default()
	router.POST("/validate-payload", handler.validatePayload)

	// Create a test request
	payload := ValidatePayloadRequest{
		Resource: "users",
		Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
	}
	requestBody := gin.H{
		"resource": payload.Resource,
		"payload":  payload.Payload,
	}
	out, _ := json.Marshal(requestBody)
	request := httptest.NewRequest(http.MethodPost, "/validate-payload", bytes.NewBuffer(out))

	// Perform the request
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// Assert the response
	assert.Equal(t, http.StatusOK, response.Code)

	var resp ValidatePayloadResponse
	json.Unmarshal(response.Body.Bytes(), &resp)
	assert.True(t, resp.Valid)
}

func TestGetSchema(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	db.Save(&wrappers.Resource{
		Model:   gorm.Model{ID: 1},
		Name:    "users",
		Schema:  "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
		Version: 1,
	}, nil)

	// Create a new HavenHandler with the fake DB
	handler := NewHavenHandler(db, nil)

	// Create a test router
	router := gin.Default()
	router.GET("/get-schema/:name", handler.getSchema)

	// Create a test request
	request := httptest.NewRequest(http.MethodGet, "/get-schema/users", nil)

	// Perform the request
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// Assert the response
	assert.Equal(t, http.StatusOK, response.Code)
	var resp GetSchemaResponse
	json.Unmarshal(response.Body.Bytes(), &resp)
	assert.Equal(t,
		map[string]any{
			"$id":                  "https://movinglake.com/haven.schema.json",
			"$schema":              "https://json-schema.org/draft/2020-12/schema",
			"additionalProperties": false,
			"properties": map[string]any{
				"age":  map[string]any{"type": "number"},
				"name": map[string]any{"type": "string"},
			},
			"required": []any{"age", "name"},
			"title":    "users",
			"type":     "object",
		},
		resp.Schema)
}

func TestSetSchema(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	// Ensure we can override the schema.
	db.Save(&wrappers.Resource{
		Model:   gorm.Model{ID: 1},
		Name:    "users",
		Schema:  "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
		Version: 1,
	}, nil)

	// Create a new HavenHandler with the fake DB
	handler := NewHavenHandler(db, nil)

	// Create a test router
	router := gin.Default()
	router.POST("/set-schema", handler.setSchema)

	// Create a test request
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "integer",
			},
		},
	}
	requestBody := gin.H{
		"resource": "users",
		"schema":   schema,
	}
	out, _ := json.Marshal(requestBody)
	request := httptest.NewRequest(http.MethodPost, "/set-schema", bytes.NewBuffer(out))

	// Perform the request
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// Assert the response
	assert.Equal(t, http.StatusOK, response.Code)
	var resp SetSchemaResponse
	json.Unmarshal(response.Body.Bytes(), &resp)
	assert.Equal(t, "users", resp.Resource.Name)
	assert.Equal(t, uint(2), resp.Resource.Version)
	assert.Equal(t, schema, resp.Resource.Schema)
}

func TestGetResources(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	// Create a new HavenHandler with the fake DB
	handler := NewHavenHandler(db, nil)

	// Create a test router
	router := gin.Default()
	router.GET("/get_all_resources", handler.getResources)

	// Create a test request
	request := httptest.NewRequest(http.MethodGet, "/get_all_resources", nil)

	// Perform the request
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// Assert the response
	assert.Equal(t, http.StatusOK, response.Code)
	var resp GetAllResourcesResponse
	json.Unmarshal(response.Body.Bytes(), &resp)
	assert.Equal(t, 0, len(resp.Resources))
}

func TestGetResourceVersions(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	// Create a new HavenHandler with the fake DB
	handler := NewHavenHandler(db, nil)

	// Create a test router
	router := gin.Default()
	router.GET("/get-resource-versions/:id", handler.getResourceVersions)

	// Create a test request
	request := httptest.NewRequest(http.MethodGet, "/get-resource-versions/1", nil)

	// Perform the request
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// Assert the response
	assert.Equal(t, http.StatusNotFound, response.Code)
	var resp GetResourceVersionsResponse
	json.Unmarshal(response.Body.Bytes(), &resp)
	assert.Equal(t, "no versions found for resource id 1", resp.Error)
}

func TestGetReferencePayload(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	// Create a new HavenHandler with the fake DB
	handler := NewHavenHandler(db, nil)

	// Create a test router
	router := gin.Default()
	router.GET("/get-reference-payload/:id", handler.getReferencePayload)

	// Create a test request
	request := httptest.NewRequest(http.MethodGet, "/get-reference-payload/1", nil)

	// Perform the request
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// Assert the response
	assert.Equal(t, http.StatusNotFound, response.Code)
	var resp GetReferencePayloadResponse
	json.Unmarshal(response.Body.Bytes(), &resp)
	assert.Equal(t, "no reference payload found for id 1", resp.Error)

}
