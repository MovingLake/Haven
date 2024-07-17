package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"movinglake.com/haven/handler"
	"movinglake.com/haven/wrappers"
)

var (
	DB_HOST = os.Getenv("DB_HOST")
	DB_USER = os.Getenv("DB_USER")
	DB_PASS = os.Getenv("DB_PASS")
	DB_NAME = os.Getenv("DB_NAME")
)

func init() {
	if DB_HOST == "" {
		DB_HOST = "localhost"
	}
	if DB_USER == "" {
		DB_USER = "postgres"
	}
	if DB_PASS == "" {
		DB_PASS = "postgres"
	}
	if DB_NAME == "" {
		DB_NAME = "haventest"
	}

}

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

func loadLoadTestData(t *testing.T) []map[string]any {
	t.Helper()
	// Load the test data from the file.
	entries, err := os.ReadDir("./testdata/loadtest")
	if err != nil {
		return nil
	}
	retval := make([]map[string]any, len(entries))
	for i, f := range entries {
		fullPath := fmt.Sprintf("./testdata/loadtest/%s", f.Name())
		out, err := os.ReadFile(fullPath)
		if err != nil {
			return nil
		}
		var data map[string]any
		err = json.Unmarshal(out, &data)
		if err != nil {
			return nil
		}
		retval[i] = data
	}
	return retval
}

func TestLoadTest(t *testing.T) {
	// Create a new Gin router
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	// Create DB connection.
	db, err := wrappers.NewDB(fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", DB_USER, DB_PASS, DB_HOST, DB_NAME))
	db.TruncateAll() // Ensure the DB is empty.
	if err != nil {
		t.Fatal(err)
	}
	h := handler.NewHavenHandler(db, nil)
	h.RegisterRoutes(router)
	payloads := loadLoadTestData(t)
	for _, p := range payloads {
		var request handler.AddPayloadRequest
		request.Payload = p
		request.Resource = "load_test"
		ser, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/add_payload", bytes.NewBuffer(ser))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("Expected status code %d but got %d %v", http.StatusOK, recorder.Code, recorder.Body)
		}
		var response handler.AddPayloadResponse
		err = json.NewDecoder(recorder.Body).Decode(&response)
		if err != nil {
			t.Fatal(err)
		}
		if response.Error != "" {
			t.Fatalf("Expected no error but got %s", response.Error)
		}
	}
	// Get resulting schema.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/load_test", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
	var response handler.GetSchemaResponse
	err = json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}
	db.TruncateAll()
	// Now run the real load test.
	errors := make(chan error, len(payloads))
	var wg sync.WaitGroup
	for _, p := range payloads {
		wg.Add(1)
		go func(p map[string]any) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				var request handler.AddPayloadRequest
				request.Payload = p
				request.Resource = "load_test"
				ser, err := json.Marshal(request)
				if err != nil {
					errors <- fmt.Errorf("Failed to marshal request: %w", err)
					return
				}
				recorder := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/add_payload", bytes.NewBuffer(ser))
				router.ServeHTTP(recorder, req)
				if recorder.Code != http.StatusOK {
					if i == 9 {
						errors <- fmt.Errorf("Failed to add payload after 10 attempts %v %v", recorder.Code, recorder.Body)
						return
					}
					time.Sleep(time.Duration(1000+rand.Int()%(1+i*1000)) * time.Millisecond)
					continue
				}
				var response handler.AddPayloadResponse
				err = json.NewDecoder(recorder.Body).Decode(&response)
				if err != nil {
					errors <- fmt.Errorf("Failed to decode response: %w body: \"%v\"", err, recorder.Body)
					return
				}
				if response.Error != "" {
					errors <- fmt.Errorf("Expected no error but got %s", response.Error)
					return
				}
				return
			}
		}(p)
	}
	wg.Wait()
	close(errors)
	for e := range errors {
		if e != nil {
			t.Error(e)
		}
	}
	// Get resulting schema.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/load_test", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}
	var response2 handler.GetSchemaResponse
	err = json.NewDecoder(recorder.Body).Decode(&response2)
	if err != nil {
		t.Fatal(err)
	}
	less := func(a, b any) bool {
		astr, ok := a.(string)
		bstr, okb := b.(string)
		if !ok || !okb {
			return false
		}
		return astr < bstr
	}
	if diff := cmp.Diff(response, response2, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("Response mismatch: %s", diff)
	}
}

func TestHealth(t *testing.T) {
	// Create a new Gin router
	router := gin.Default()
	gin.SetMode(gin.TestMode)
	// Create DB connection.
	db, err := wrappers.NewDB(fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", DB_USER, DB_PASS, DB_HOST, DB_NAME))
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
	gin.SetMode(gin.TestMode)
	// Create DB connection.
	db, err := wrappers.NewDB(fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", DB_USER, DB_PASS, DB_HOST, DB_NAME))
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
	gin.SetMode(gin.TestMode)
	// Create DB connection.
	db, err := wrappers.NewDB(fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", DB_USER, DB_PASS, DB_HOST, DB_NAME))
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
	var setSchemaResponse handler.SetSchemaResponse
	if err := json.NewDecoder(recorder.Body).Decode(&setSchemaResponse); err != nil {
		t.Fatal(err)
	}
	if setSchemaResponse.Error != "" || setSchemaResponse.Success != true {
		t.Fatalf("Expected no error but got %s", setSchemaResponse.Error)
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
		t.Fatalf("Expected 2 resources but got %v", allResp.Resources)
	}

	// Get all versions of a resource.
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_resource_versions/"+strconv.Itoa(int(addPayloadResponse.Resource.ID)), nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d %v", http.StatusOK, recorder.Code, recorder.Body)
	}
	var versionsResp handler.GetResourceVersionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&versionsResp); err != nil {
		t.Fatal(err)
	}
	if versionsResp.Error != "" {
		t.Fatalf("Expected no error but got %s", versionsResp.Error)
	}
	if len(versionsResp.Versions) != 2 {
		t.Fatalf("Expected 2 versions but got %v", versionsResp)
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

func TestBadRequests(t *testing.T) {
	// Create a new Gin router
	router := gin.Default()
	gin.SetMode(gin.TestMode)
	// Create DB connection.
	db, err := wrappers.NewDB(fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", DB_USER, DB_PASS, DB_HOST, DB_NAME))
	db.TruncateAll() // Ensure the DB is empty.
	if err != nil {
		t.Fatal(err)
	}
	h := handler.NewHavenHandler(db, nil)
	h.RegisterRoutes(router)
	db.Save(&wrappers.Resource{
		Name:    "test",
		Schema:  "{\"type\": \"object\"}",
		Version: 1,
	}, nil)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/test", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_schema", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/3klj45@##", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_schema/3klj45/@##", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_resource_versions/23", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/get_resource_versions/notfound", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
