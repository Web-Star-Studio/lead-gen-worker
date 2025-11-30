package api

import (
	"net/http"

	"webstar/noturno-leadgen-worker/internal/api/controllers"
	"webstar/noturno-leadgen-worker/internal/handlers"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter creates and configures a new Gin router
func NewRouter(searchHandler *handlers.GoogleSearchHandler) *gin.Engine {
	router := gin.Default() // Includes Logger and Recovery middleware

	// Initialize controllers
	searchController := controllers.NewSearchController(searchHandler)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Swagger documentation route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/search", searchController.Search)
	}

	return router
}
