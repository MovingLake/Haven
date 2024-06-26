package lib

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *HavenHandler) Router(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if v, ok := payload["action"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no action specified"})
		return
	} else {
		switch v {
		case "addPayload":
			if err := h.addPayload(payload); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action " + v.(string)})
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// addPayload adds a new payload to the specific resource.
func (h *HavenHandler) addPayload(p hjson) error {
	resource := p["resource"].(string)
	payload := p["payload"].(hjson)

	// Get the schema of the resource.
	t := h.db.OpenTxn()
	r := &wrappers.Resource{}
	t.Find(r, "name = ?", resource)
	if r.ID == 0 {
		return fmt.Errorf("resource not found for %s", resource)
	}

	schema := make(hjson)
	if err := json.Unmarshal([]byte(r.Schema), &schema); err != nil {
		return err
	}

	diffs, newSchema, err := computeSchemaDiff(schema, payload)

	return nil
}

func (h *HavenHandler) RegisterRoutes(e *gin.Engine) error {
	e.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})
	e.POST("/api", h.Router)
	return nil
}
