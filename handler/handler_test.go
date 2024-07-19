package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"movinglake.com/haven/handler/notifications"
	"movinglake.com/haven/wrappers"
)

type fakeSlackSender struct {
	err error
}

func (f *fakeSlackSender) SendMessage(message string) error {
	return f.err
}

func (f *fakeSlackSender) IsActive() bool {
	return true
}

func TestAddPayload(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB().(*wrappers.TestDB)

	cases := []struct {
		name       string
		dbErrors   map[string]error
		dbResource *wrappers.Resource
		request    *AddPayloadRequest
		rawRequest string
		slacker    notifications.Sender
		want       *AddPayloadResponse
		wantCode   int
	}{
		{
			name: "Slack gets sent",
			request: &AddPayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
			},
			slacker:  &fakeSlackSender{},
			wantCode: http.StatusOK,
			want: &AddPayloadResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 1,
				},
			},
		},
		{
			name:     "DB Save Fails",
			dbErrors: map[string]error{"Save": gorm.ErrInvalidData},
			request: &AddPayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:       "Payload is not parsable, no resource",
			rawRequest: "{\"resource\": \"users\", \"payload\": {\"name: \"John Doe\", \"age\": 30}",
			wantCode:   http.StatusBadRequest,
		},
		{
			name: "Payload is not parsable, resource exists",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "{\"type\": \"object\", \"additionalProperties\": false}",
				Version: 1,
			},
			rawRequest: "{\"resource\": \"users\", \"payload\": {\"name: \"John Doe\", \"age\": 30}",
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "Request malformed",
			rawRequest: "Some { miss formated \"\"]} payload",
			wantCode:   http.StatusBadRequest,
		},
		{
			name: "DB contains bad schema",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "some non <json> schema",
				Version: 1,
			},
			request: &AddPayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "DB failed to find",
			dbErrors: map[string]error{
				"SelectResourceForUpdate": gorm.ErrRecordNotFound,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "valid request no resource",
			request: &AddPayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
			},
			want: &AddPayloadResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 1,
				},
			},
			wantCode: http.StatusOK,
		},
		{
			name: "valid request existing resource",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "{\"type\": \"object\", \"additionalProperties\": false}",
				Version: 1,
			},
			request: &AddPayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
			},
			want: &AddPayloadResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
						"additionalProperties": false,
						"properties": map[string]any{
							"age":  map[string]any{"type": "number"},
							"name": map[string]any{"type": "string"},
						},
						"type": "object",
					},
					Version: 2,
				},
			},
			wantCode: http.StatusOK,
		},
		{
			name: "valid request existing resource no schema change",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "{\"type\": \"object\", \"additionalProperties\": false}",
				Version: 1,
			},
			request: &AddPayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{},
			},
			want: &AddPayloadResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
						"type":                 "object",
						"additionalProperties": false,
					},
					Version: 1,
				},
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new HavenHandler with the fake DB
			handler := NewHavenAPIHandler(db, nil)
			handler.slacker = tc.slacker

			// Create a test router
			router := gin.Default()
			gin.SetMode(gin.TestMode)
			handler.RegisterRoutes(router)
			db.Errors = nil
			if err := db.TruncateAll(); err != nil {
				t.Fatalf("Failed to truncate db %v", err)
			}
			if err := db.Save(tc.dbResource, nil); err != nil {
				t.Fatalf("Failed to save resource %v %v", tc.dbResource, err)
			}
			db.Errors = tc.dbErrors

			out, _ := json.Marshal(tc.request)
			if tc.rawRequest != "" {
				out = []byte(tc.rawRequest)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/add_payload", bytes.NewBuffer(out))

			// Perform the request
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			// Assert the response
			assert.Equal(t, tc.wantCode, response.Code)
			if tc.wantCode != http.StatusOK {
				return
			}
			resp := &AddPayloadResponse{}
			json.Unmarshal(response.Body.Bytes(), resp)
			assert.Equal(t, tc.want.Success, resp.Success)
			ignoreTimeFields := cmpopts.IgnoreFields(ResourceResp{}, "CreatedAt", "UpdatedAt")
			if dif := cmp.Diff(tc.want.Resource, resp.Resource, ignoreTimeFields); dif != "" {
				t.Errorf("AddPayload(%v) got a diff: %s", tc.request, dif)
			}
		})
	}
}

func TestValidatePayload(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB().(*wrappers.TestDB)

	// Create a new HavenHandler with the fake DB
	handler := NewHavenAPIHandler(db, nil)

	// Create a test router
	router := gin.Default()
	gin.SetMode(gin.TestMode)
	handler.RegisterRoutes(router)

	cases := []struct {
		name       string
		dbErrors   map[string]error
		dbResource *wrappers.Resource
		request    *ValidatePayloadRequest
		want       *ValidatePayloadResponse
		wantCode   int
	}{
		{
			name: "valid",
			dbResource: &wrappers.Resource{
				Model:   gorm.Model{ID: 1},
				Name:    "users",
				Schema:  "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
				Version: 1,
			},
			request: &ValidatePayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{"name": "John Doe", "age": 30},
			},
			want: &ValidatePayloadResponse{
				Valid: true,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "invalid",
			dbResource: &wrappers.Resource{
				Model:   gorm.Model{ID: 1},
				Name:    "users",
				Schema:  "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
				Version: 1,
			},
			request: &ValidatePayloadRequest{
				Resource: "users",
				Payload:  map[string]interface{}{"narnia": "ok", "name": "Juan", "age": 35},
			},
			want: &ValidatePayloadResponse{
				Valid: false,
				ValidationErrors: []ErrorResponse{
					{
						Type:        "additional_property_not_allowed",
						Description: "Additional property narnia is not allowed",
						Context: map[string]any{
							"expected": nil,
							"field":    "(root)",
							"given":    nil,
							"path":     "(root).",
							"property": "narnia",
						},
					},
				},
			},
			wantCode: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.TruncateAll(); err != nil {
				t.Fatalf("Failed to truncate db %v", err)
			}
			if err := db.Save(tc.dbResource, nil); err != nil {
				t.Fatalf("Failed to save resource %v %v", tc.dbResource, err)
			}
			db.Errors = tc.dbErrors
			out, _ := json.Marshal(tc.request)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/validate_payload", bytes.NewBuffer(out))

			// Perform the request
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			// Assert the response
			assert.Equal(t, tc.wantCode, response.Code)
			if tc.wantCode != http.StatusOK {
				return
			}

			var resp ValidatePayloadResponse
			json.Unmarshal(response.Body.Bytes(), &resp)
			if diff := cmp.Diff(tc.want, &resp); diff != "" {
				t.Errorf("ValidatePayload(%v) got a diff: %s", tc.request, diff)
			}
		})
	}
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
	handler := NewHavenAPIHandler(db, nil)

	// Create a test router
	router := gin.Default()
	gin.SetMode(gin.TestMode)
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
	db := wrappers.NewTestDB().(*wrappers.TestDB)
	handler := NewHavenAPIHandler(db, nil)
	router := gin.Default()
	gin.SetMode(gin.TestMode)
	handler.RegisterRoutes(router)

	cases := []struct {
		name       string
		dbErrors   map[string]error
		dbResource *wrappers.Resource
		request    *SetSchemaRequest
		want       *SetSchemaResponse
		wantCode   int
	}{
		{
			name: "DB failed",
			dbErrors: map[string]error{
				"GetResource": gorm.ErrDuplicatedKey,
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "valid request no resource",
			request: &SetSchemaRequest{
				Resource: "users",
				Schema: map[string]any{
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
			},
			want: &SetSchemaResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 1,
				},
			},
			wantCode: http.StatusOK,
		},
		{
			name: "valid request existing resource",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "{\"type\": \"object\", \"additionalProperties\": false}",
				Version: 1,
			},
			request: &SetSchemaRequest{
				Resource: "users",
				Schema: map[string]any{
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
			},
			want: &SetSchemaResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 2,
				},
			},
			wantCode: http.StatusOK,
		},
		{
			name: "valid request existing empty schema resource",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "",
				Version: 1,
			},
			request: &SetSchemaRequest{
				Resource: "users",
				Schema: map[string]any{
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
			},
			want: &SetSchemaResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 2,
				},
			},
			wantCode: http.StatusOK,
		},
		{
			name: "valid request existing empty obj schema resource",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "{}",
				Version: 1,
			},
			request: &SetSchemaRequest{
				Resource: "users",
				Schema: map[string]any{
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
			},
			want: &SetSchemaResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 2,
				},
			},
			wantCode: http.StatusOK,
		},
		{
			name: "valid request existing malformed schema resource",
			dbResource: &wrappers.Resource{
				Name:    "users",
				Schema:  "some non <json> schema",
				Version: 1,
			},
			request: &SetSchemaRequest{
				Resource: "users",
				Schema: map[string]any{
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
			},
			want: &SetSchemaResponse{
				Success: true,
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 2,
				},
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.TruncateAll(); err != nil {
				t.Fatalf("Failed to truncate db %v", err)
			}
			if err := db.Save(tc.dbResource, nil); err != nil {
				t.Fatalf("Failed to save resource %v %v", tc.dbResource, err)
			}
			db.Errors = tc.dbErrors
			out, _ := json.Marshal(tc.request)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/set_schema", bytes.NewBuffer(out))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			assert.Equal(t, tc.wantCode, response.Code)
			if tc.wantCode != http.StatusOK {
				return
			}
			var resp SetSchemaResponse
			json.Unmarshal(response.Body.Bytes(), &resp)
			ignoreTimeFields := cmpopts.IgnoreFields(ResourceResp{}, "CreatedAt", "UpdatedAt")
			if diff := cmp.Diff(tc.want, &resp, ignoreTimeFields); diff != "" {
				t.Errorf("SetSchema(%v) got a diff: %s", tc.request, diff)
			}
		})
	}
}

func TestGetResources(t *testing.T) {
	// Create a fake DB
	db := wrappers.NewTestDB()

	// Create a new HavenHandler with the fake DB
	handler := NewHavenAPIHandler(db, nil)

	// Create a test router
	router := gin.Default()
	gin.SetMode(gin.TestMode)
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
	handler := NewHavenAPIHandler(db, nil)

	// Create a test router
	router := gin.Default()
	gin.SetMode(gin.TestMode)
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
	handler := NewHavenAPIHandler(db, nil)

	// Create a test router
	router := gin.Default()
	gin.SetMode(gin.TestMode)
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

func TestBadPostRequests(t *testing.T) {
	db := wrappers.NewTestDB()
	handler := NewHavenAPIHandler(db, nil)

	router := gin.Default()
	gin.SetMode(gin.TestMode)
	handler.RegisterRoutes(router)

	requestBody := "Some { miss formated \"\"]} payload"
	out, _ := json.Marshal(requestBody)

	// POST requests.
	request := httptest.NewRequest(http.MethodPost, "/api/v1/set_schema", bytes.NewBuffer(out))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusBadRequest, response.Code)

	request = httptest.NewRequest(http.MethodPost, "/api/v1/add_payload", bytes.NewBuffer(out))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusBadRequest, response.Code)

	request = httptest.NewRequest(http.MethodPost, "/api/v1/validate_payload", bytes.NewBuffer(out))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestGetResource(t *testing.T) {
	db := wrappers.NewTestDB().(*wrappers.TestDB)
	handler := NewHavenAPIHandler(db, nil)
	router := gin.Default()
	gin.SetMode(gin.TestMode)
	handler.RegisterRoutes(router)

	cases := []struct {
		name       string
		dbErrors   map[string]error
		dbResource *wrappers.Resource
		request    string
		want       *GetResourceResponse
		wantCode   int
	}{
		{
			name: "DB failed",
			dbErrors: map[string]error{
				"GetResource": gorm.ErrDuplicatedKey,
			},
			request:  "users",
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "valid request",
			dbResource: &wrappers.Resource{
				Model:   gorm.Model{ID: 1},
				Name:    "users",
				Schema:  "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
				Version: 1,
			},
			request: "users",
			want: &GetResourceResponse{
				Resource: ResourceResp{
					ID:   1,
					Name: "users",
					Schema: map[string]any{
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
					Version: 1,
				},
			},
			wantCode: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.TruncateAll(); err != nil {
				t.Fatalf("Failed to truncate db %v", err)
			}
			if err := db.Save(tc.dbResource, nil); err != nil {
				t.Fatalf("Failed to save resource %v %v", tc.dbResource, err)
			}
			db.Errors = tc.dbErrors
			request := httptest.NewRequest(http.MethodGet, "/api/v1/get_resource/"+tc.request, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			assert.Equal(t, tc.wantCode, response.Code)
			if tc.wantCode != http.StatusOK {
				return
			}
			var resp GetResourceResponse
			json.Unmarshal(response.Body.Bytes(), &resp)
			ignoreTimeFields := cmpopts.IgnoreFields(ResourceResp{}, "CreatedAt", "UpdatedAt")
			if diff := cmp.Diff(tc.want, &resp, ignoreTimeFields); diff != "" {
				t.Errorf("GetResource(%v) got a diff: %s", tc.request, diff)
			}
		})
	}
}

func TestGetResourceVersion(t *testing.T) {
	db := wrappers.NewTestDB().(*wrappers.TestDB)
	handler := NewHavenAPIHandler(db, nil)
	router := gin.Default()
	gin.SetMode(gin.TestMode)
	handler.RegisterRoutes(router)
	one := 1

	cases := []struct {
		name      string
		dbErrors  map[string]error
		dbVersion *wrappers.ResourceVersions
		requestID int
		want      *GetResourceVersionResponse
		wantCode  int
	}{
		{
			name: "DB failed",
			dbErrors: map[string]error{
				"GetResourceVersion": gorm.ErrDuplicatedKey,
			},
			requestID: 1,
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:      "version not found",
			requestID: 1,
			wantCode:  http.StatusNotFound,
		},
		{
			name: "valid request",
			dbVersion: &wrappers.ResourceVersions{
				Model:              gorm.Model{ID: 1},
				Version:            1,
				ResourceID:         1,
				ReferencePayloadID: &one,
				OldSchema:          "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
				NewSchema:          "{\"$id\":\"https://movinglake.com/haven.schema.json\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"additionalProperties\":false,\"properties\":{\"age\":{\"type\":\"number\"},\"name\":{\"type\":\"string\"}},\"required\":[\"age\",\"name\"],\"title\":\"users\",\"type\":\"object\"}",
			},
			requestID: 1,
			want: &GetResourceVersionResponse{
				Version: ResourceVersionsResponse{
					ID:               1,
					Version:          1,
					Resource:         1,
					ReferencePayload: 0,
					OldSchema: map[string]any{
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
					NewSchema: map[string]any{
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
				},
			},
			wantCode: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.TruncateAll(); err != nil {
				t.Fatalf("Failed to truncate db %v", err)
			}
			if err := db.Save(tc.dbVersion, nil); err != nil {
				t.Fatalf("Failed to save resource %v %v", tc.dbVersion, err)
			}
			db.Errors = tc.dbErrors

			request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("%s/%d", "/api/v1/get_resource_version", tc.requestID), nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			assert.Equal(t, tc.wantCode, response.Code)
			if tc.wantCode != http.StatusOK {
				return
			}
			var resp GetResourceVersionResponse
			json.Unmarshal(response.Body.Bytes(), &resp)
			if diff := cmp.Diff(tc.want, &resp); diff != "" {
				t.Errorf("GetResourceVersion(%v) got a diff: %s", tc.requestID, diff)
			}
		})
	}
}
