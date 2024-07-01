package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"movinglake.com/haven/handler"
	"movinglake.com/haven/wrappers"
)

func main() {
	// Load dotenv.
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get vars from env.
	dbStr := os.Getenv("DB_STR")
	if dbStr == "" {
		log.Fatal("DB_STR not set")
	}

	// Create DB connection.
	db, err := wrappers.NewDB(dbStr)

	handler := handler.NewHavenHandler(db)

	r := gin.Default()
	handler.RegisterRoutes(r)
	r.Run() // listen and serve on 0.0.0.0:8080
}
