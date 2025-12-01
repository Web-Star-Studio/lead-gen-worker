package main

import (
	"log"

	"webstar/noturno-leadgen-worker/internal/api"
	"webstar/noturno-leadgen-worker/internal/api/controllers"
	"webstar/noturno-leadgen-worker/internal/config"
	"webstar/noturno-leadgen-worker/internal/handlers"
	"webstar/noturno-leadgen-worker/internal/services"

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

	// Initialize SupabaseHandler if credentials are configured
	var supabaseHandler *handlers.SupabaseHandler
	if cfg.SupabaseURL != "" && cfg.SupabaseKey != "" {
		var err error
		supabaseHandler, err = handlers.NewSupabaseHandler(cfg.SupabaseURL, cfg.SupabaseKey)
		if err != nil {
			log.Printf("Warning: Failed to initialize SupabaseHandler: %v", err)
			log.Printf("Continuing without Supabase functionality")
		} else {
			log.Printf("SupabaseHandler initialized - database access enabled")
		}
	} else {
		log.Printf("SUPABASE_URL or SUPABASE_SECRET_KEY not set - database access disabled")
	}

	// Initialize JobProcessor and WebhookController if Supabase and webhook secret are configured
	var webhookController *controllers.WebhookController
	if supabaseHandler != nil && cfg.WebhookSecret != "" {
		jobProcessor := services.NewJobProcessor(supabaseHandler, searchHandler)
		webhookController = controllers.NewWebhookController(cfg.WebhookSecret, jobProcessor)
		log.Printf("WebhookController initialized - job webhook endpoint enabled")
	} else {
		if supabaseHandler == nil {
			log.Printf("SupabaseHandler not initialized - webhook endpoint disabled")
		}
		if cfg.WebhookSecret == "" {
			log.Printf("WEBHOOK_SECRET not set - webhook endpoint disabled")
		}
	}

	// Initialize DataExtractorHandler if Google API key or Vertex AI is configured
	if cfg.GoogleAPIKey != "" || cfg.UseVertexAI {
		// Debug: log first 10 chars of API key to verify correct key is loaded
		if len(cfg.GoogleAPIKey) > 10 {
			log.Printf("[DEBUG] GOOGLE_API_KEY loaded: %s...", cfg.GoogleAPIKey[:10])
		}
		dataExtractorHandler, err := handlers.NewDataExtractorHandler(handlers.DataExtractorConfig{
			APIKey:      cfg.GoogleAPIKey,
			UseVertexAI: cfg.UseVertexAI,
			GCPProject:  cfg.GCPProject,
			GCPLocation: cfg.GCPLocation,
			// Use flash model for faster extraction
			Model: handlers.DefaultExtractorModel,
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize DataExtractorHandler: %v", err)
			log.Printf("Continuing without data extraction functionality")
		} else {
			searchHandler.SetDataExtractorHandler(dataExtractorHandler)
			backend := "Google AI Studio"
			if cfg.UseVertexAI {
				backend = "Vertex AI"
			}
			log.Printf("DataExtractorHandler initialized - data extraction enabled (backend: %s, model: %s)",
				backend, handlers.DefaultExtractorModel)
		}
	} else {
		log.Printf("GOOGLE_API_KEY or Vertex AI not configured - data extraction disabled")
	}

	// Initialize PreCallReportHandler if Google API key or Vertex AI is configured
	if cfg.GoogleAPIKey != "" || cfg.UseVertexAI {
		preCallReportHandler, err := handlers.NewPreCallReportHandler(handlers.PreCallReportConfig{
			APIKey:      cfg.GoogleAPIKey,
			Model:       cfg.GeminiModel,
			UseVertexAI: cfg.UseVertexAI,
			GCPProject:  cfg.GCPProject,
			GCPLocation: cfg.GCPLocation,
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize PreCallReportHandler: %v", err)
			log.Printf("Continuing without pre-call report generation")
		} else {
			searchHandler.SetPreCallReportHandler(preCallReportHandler)
			backend := "Google AI Studio"
			if cfg.UseVertexAI {
				backend = "Vertex AI"
			}
			model := cfg.GeminiModel
			if model == "" {
				model = handlers.DefaultGeminiModel
			}
			log.Printf("PreCallReportHandler initialized - pre-call report generation enabled (backend: %s, model: %s)",
				backend, model)
		}
	} else {
		log.Printf("GOOGLE_API_KEY or Vertex AI not configured - pre-call report generation disabled")
	}

	// Setup router
	router := api.NewRouter(searchHandler, webhookController)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
