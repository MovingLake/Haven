package handler

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"movinglake.com/haven/wrappers"
)

type HavenHTMLHandler struct {
	db wrappers.DB
}

func NewHavenHTMLHandler(db wrappers.DB) *HavenHTMLHandler {
	return &HavenHTMLHandler{
		db: db,
	}
}

func (h *HavenHTMLHandler) home(c *gin.Context) {
	resources, err := h.db.GetAllResources()
	if err != nil {
		log.Printf("Error getting resources: %v", err)
	}
	var formattedResources []string
	for _, r := range resources {
		formattedResources = append(formattedResources, r.Name)
	}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":     "Haven",
		"resources": formattedResources,
		"config":    "",
		"logs":      "",
		"metrics":   "",
	})
}

func (h *HavenHTMLHandler) RegisterRoutes(r *gin.Engine, templateRegex string, staticDir string) {
	r.LoadHTMLGlob(templateRegex)
	r.GET("/", h.home)
	r.GET("/index", h.home)
	r.GET("/index.html", h.home)
	r.GET("/resource/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.HTML(http.StatusOK, "resource.html", gin.H{
			"title":         "Haven",
			"resource_name": name,
		})
	})
	r.GET("/js/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.File(filepath.Join(staticDir, "js", name))
	})
	r.GET("/css/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.File(filepath.Join(staticDir, "css", name))
	})
	r.GET("/img/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.File(filepath.Join(staticDir, "img", name))
	})
}
