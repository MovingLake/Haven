package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"movinglake.com/haven/wrappers"
)

func TestNewHavenHTMLHandler(t *testing.T) {
	db := wrappers.NewTestDB()
	handler := NewHavenHTMLHandler(db)

	assert.NotNil(t, handler)
	assert.Equal(t, db, handler.db)
}

func TestRegisterRoutes(t *testing.T) {
	db := wrappers.NewTestDB()
	handler := NewHavenHTMLHandler(db)

	router := gin.Default()
	gin.SetMode(gin.TestMode)
	handler.RegisterRoutes(router, "../templates/*", "../web_resources")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/index", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	db.Save(&wrappers.Resource{
		Model:   gorm.Model{ID: 1},
		Name:    "some",
		Schema:  "{json-schema}",
		Version: 1,
	}, nil)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/resource/some", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/js/jsonTree.js", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/css/jsonTree.css", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/img/jsonTree.svg", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
