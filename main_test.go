package main_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"movinglake.com/haven/handler"
	"movinglake.com/haven/wrappers"
)

func TestIntegration(t *testing.T) {
	// Create a new Gin router
	router := gin.Default()
	// Create DB connection.
	db, err := wrappers.NewDB("postgresql://user:password@localhost:5432/haven?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	handler := handler.NewHavenHandler(db)

	// Create a response recorder to record the response
	recorder := httptest.NewRecorder()

	// Serve the HTTP request using the router and response recorder
	router.ServeHTTP(recorder, req)

	// Check the response status code
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}

	// Check the response body
	expectedBody := `{"message":"Hello, world!"}`
	if recorder.Body.String() != expectedBody {
		t.Errorf("Expected body %s but got %s", expectedBody, recorder.Body.String())
	}
}
