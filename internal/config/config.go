package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port            string
	SerpAPIKey      string
	FirecrawlAPIKey string
	FirecrawlAPIURL string // Optional: custom Firecrawl API URL (leave empty for default)
}

// Load reads configuration from environment variables
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:            port,
		SerpAPIKey:      os.Getenv("SERPAPI_KEY"),
		FirecrawlAPIKey: os.Getenv("FIRECRAWL_API_KEY"),
		FirecrawlAPIURL: os.Getenv("FIRECRAWL_API_URL"), // Optional
	}
}
