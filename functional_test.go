package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"movinglake.com/haven/handler"
	"movinglake.com/haven/wrappers"
)

type TestData struct {
	Requests []handler.AddPayloadRequest `json:"requests"`
}

func loadTestDataSchema(t *testing.T, file string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func loadTestData(t *testing.T, file string) TestData {
	t.Helper()
	fmt.Println("Loading test data from", file)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var testJson TestData
	if err := json.Unmarshal(data, &testJson); err != nil {
		t.Fatal(err)
	}
	return testJson
}

func TestHealth(t *testing.T) {
	// Create a new Gin router
	router := gin.Default()
	// Create DB connection.
	db, err := wrappers.NewDB("postgresql://localhost:5432/haventest?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	handler := handler.NewHavenHandler(db, &handler.NotificationsConfig{})
	handler.RegisterRoutes(router)

	// Create a response recorder to record the response
	recorder := httptest.NewRecorder()

	// Create a new HTTP request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	// Serve the HTTP request using the router and response recorder
	router.ServeHTTP(recorder, req)

	// Check the response status code
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
}

func TestAddPayload(t *testing.T) {
	// Create a new Gin router
	router := gin.Default()
	// Create DB connection.
	db, err := wrappers.NewDB("postgresql://localhost:5432/haventest?sslmode=disable")
	db.TruncateAll() // Ensure the DB is empty.
	if err != nil {
		t.Fatal(err)
	}
	h := handler.NewHavenHandler(db, nil)
	h.RegisterRoutes(router)

	cases := []struct {
		name       string
		testFile   string
		wantSchema string
	}{
		{
			name:       "single_payload",
			testFile:   "testdata/single_payload.json",
			wantSchema: "testdata/single_payload_schema.json",
		},
		{
			name:       "multiple_payloads",
			testFile:   "testdata/multiple_payloads.json",
			wantSchema: "testdata/multiple_payloads_schema.json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a response recorder to record the response
			recorder := httptest.NewRecorder()

			// Load the test file.
			testData := loadTestData(t, tc.testFile)

			// Send all payloads to the API.
			for _, req := range testData.Requests {
				ser, err := json.Marshal(req)
				if err != nil {
					t.Fatal(err)
				}

				// Create a new HTTP request
				req := httptest.NewRequest(http.MethodPost, "/api/v1/add_payload", bytes.NewBuffer(ser))

				// Serve the HTTP request using the router and response recorder
				router.ServeHTTP(recorder, req)

				// Check the response status code
				if recorder.Code != http.StatusOK {
					t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
				}

				// Check the response body
				var response handler.AddPayloadResponse
				if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
					t.Fatal(err)
				}
				if response.Error != "" {
					t.Fatalf("Expected no error but got %s", response.Error)
				}
				if response.Success != true {
					t.Fatalf("Expected message 'ok' but got %v", response)
				}
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/test", nil)
			// Serve the HTTP request using the router and response recorder
			router.ServeHTTP(recorder, req)

			// Check the response status code
			if recorder.Code != http.StatusOK {
				t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
			}
			var response handler.GetSchemaResponse
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.Error != "" {
				t.Fatalf("Expected no error but got %s", response.Error)
			}
			wantSchema := loadTestDataSchema(t, tc.wantSchema)
			if diff := cmp.Diff(wantSchema, response.Schema); diff != "" {
				t.Fatalf("Schemas don't match (-want, +got)\n%v", diff)
			}

			// Now the schema should be valid for all payloads.
			for _, req := range testData.Requests {
				vReq := &handler.ValidatePayloadRequest{
					Resource: "test",
					Payload:  req.Payload,
				}
				ser, err := json.Marshal(vReq)
				if err != nil {
					t.Fatal(err)
				}

				// Create a new HTTP request
				req := httptest.NewRequest(http.MethodPost, "/api/v1/validate_payload", bytes.NewBuffer(ser))

				// Serve the HTTP request using the router and response recorder
				router.ServeHTTP(recorder, req)

				// Check the response status code
				if recorder.Code != http.StatusOK {
					t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
				}

				// Check the response body
				var response handler.ValidatePayloadResponse
				if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
					t.Fatal(err)
				}
				if response.Error != "" {
					t.Fatalf("Expected no error but got %s", response.Error)
				}
				if response.Valid != true {
					t.Fatalf("Expected message 'ok' but got %v", response)
				}
			}
		})
	}
}

func TestCruds(t *testing.T) {
	// Create a new Gin router
	router := gin.Default()
	// Create DB connection.
	db, err := wrappers.NewDB("postgresql://localhost:5432/haventest?sslmode=disable")
	db.TruncateAll() // Ensure the DB is empty.
	if err != nil {
		t.Fatal(err)
	}
	h := handler.NewHavenHandler(db, nil)
	h.RegisterRoutes(router)

	recorder := httptest.NewRecorder()

	// Set Schema.
	setSchemaReq := handler.SetSchemaRequest{
		Resource: "test",
		Schema:   map[string]any{"type": "object", "additionalProperties": false},
	}
	ser, err := json.Marshal(setSchemaReq)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/set_schema", bytes.NewBuffer(ser))
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d %s", http.StatusOK, recorder.Code, recorder.Body)
	}

	// Get Schema.
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/test", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
	var response handler.GetSchemaResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "" {
		t.Fatalf("Expected no error but got %s", response.Error)
	}
	wantSchema := map[string]any{"type": "object", "additionalProperties": false}
	if diff := cmp.Diff(wantSchema, response.Schema); diff != "" {
		t.Fatalf("Schemas don't match (-want, +got)\n%v", diff)
	}

	// Add Payload.
	recorder = httptest.NewRecorder()
	addPayloadReq := handler.AddPayloadRequest{
		Resource: "test",
		Payload:  map[string]any{"name": "test", "age": 10},
	}
	ser, err = json.Marshal(addPayloadReq)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/add_payload", bytes.NewBuffer(ser))
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
	var addPayloadResponse handler.AddPayloadResponse
	if err := json.NewDecoder(recorder.Body).Decode(&addPayloadResponse); err != nil {
		t.Fatal(err)
	}
	if addPayloadResponse.Error != "" {
		t.Fatalf("Expected no error but got %s", addPayloadResponse.Error)
	}
	if addPayloadResponse.Success != true {
		t.Fatalf("Expected message 'ok' but got %v", addPayloadResponse)
	}

	// Set a second Schema.
	recorder = httptest.NewRecorder()
	setSchemaReq = handler.SetSchemaRequest{
		Resource: "test2",
		Schema:   map[string]any{"type": "object"},
	}
	ser, err = json.Marshal(setSchemaReq)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/set_schema", bytes.NewBuffer(ser))
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}

	// Get all resources.
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_all_resources", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
	var allResp handler.GetAllResourcesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&allResp); err != nil {
		t.Fatal(err)
	}
	if allResp.Error != "" {
		t.Fatalf("Expected no error but got %s", allResp.Error)
	}
	if len(allResp.Resources) != 2 {
		t.Fatalf("Expected 2 resources but got %d", len(allResp.Resources))
	}

	// Get all versions of a resource.
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_resource_versions/1", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
	var versionsResp handler.GetResourceVersionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&versionsResp); err != nil {
		t.Fatal(err)
	}
	if versionsResp.Error != "" {
		t.Fatalf("Expected no error but got %s", versionsResp.Error)
	}
	if len(versionsResp.Versions) != 2 {
		t.Fatalf("Expected 2 versions but got %d", len(versionsResp.Versions))
	}

	// Get a reference payload.
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_reference_payload/1", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
	var refPayload handler.GetReferencePayloadResponse
	if err := json.NewDecoder(recorder.Body).Decode(&refPayload); err != nil {
		t.Fatal(err)
	}
	if refPayload.Error != "" {
		t.Fatalf("Expected no error but got %s", refPayload.Error)
	}
	if refPayload.Payload == nil {
		t.Fatalf("Expected a payload but got nil")
	}

}
