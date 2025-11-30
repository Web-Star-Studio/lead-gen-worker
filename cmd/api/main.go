package main

import (
	"log"

	"webstar/noturno-leadgen-worker/internal/api"
	"webstar/noturno-leadgen-worker/internal/config"
	"webstar/noturno-leadgen-worker/internal/handlers"

	_ "webstar/noturno-leadgen-worker/docs" // Swagger generated docs
)

// @title Lead Gen Worker API
// @version 1.0
// @description A high-performance REST API service for scraping Google search results using SerpAPI. Designed for lead generation workflows.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @schemes http https
func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Validate required configuration
	if cfg.SerpAPIKey == "" {
		log.Fatal("SERPAPI_KEY environment variable is required")
	}

	// Initialize handlers
	searchHandler := handlers.NewGoogleSearchHandler(cfg.SerpAPIKey)

	// Initialize FirecrawlHandler if API key is configured
	if cfg.FirecrawlAPIKey != "" {
		firecrawlHandler, err := handlers.NewFirecrawlHandler(cfg.FirecrawlAPIKey, cfg.FirecrawlAPIURL)
		if err != nil {
			log.Printf("Warning: Failed to initialize FirecrawlHandler: %v", err)
			log.Printf("Continuing without website scraping functionality")
		} else {
			searchHandler.SetFirecrawlHandler(firecrawlHandler)
			log.Printf("FirecrawlHandler initialized - website scraping enabled")
		}
	} else {
		log.Printf("FIRECRAWL_API_KEY not set - website scraping disabled")
	}

	// Setup router
	router := api.NewRouter(searchHandler)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
