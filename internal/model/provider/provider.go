// Package provider provides a unified interface for creating LLM models
// supporting both Google Gemini and OpenRouter backends.
package provider

import (
	"context"
	"fmt"
	"log"
	"os"

	"webstar/noturno-leadgen-worker/internal/model/openrouter"

	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

// Backend represents the LLM backend to use
type Backend string

const (
	// BackendGemini uses Google AI Studio (Gemini API)
	BackendGemini Backend = "gemini"
	// BackendVertexAI uses Google Cloud Vertex AI
	BackendVertexAI Backend = "vertexai"
	// BackendOpenRouter uses OpenRouter API
	BackendOpenRouter Backend = "openrouter"
)

// Config holds configuration for creating an LLM model
type Config struct {
	// Backend specifies which LLM backend to use
	Backend Backend

	// Model name (required)
	// For Gemini: "gemini-2.5-flash", "gemini-2.5-pro", etc.
	// For OpenRouter: "anthropic/claude-3.5-sonnet", "openai/gpt-4o", etc.
	Model string

	// Google AI Studio configuration
	GoogleAPIKey string

	// Vertex AI configuration
	GCPProject  string
	GCPLocation string

	// OpenRouter configuration
	OpenRouterAPIKey   string
	OpenRouterBaseURL  string
	OpenRouterSiteURL  string // For OpenRouter rankings (HTTP-Referer)
	OpenRouterSiteName string // For OpenRouter rankings (X-Title)
}

// NewModel creates a new LLM model based on the configuration
func NewModel(ctx context.Context, cfg Config) (model.LLM, error) {
	switch cfg.Backend {
	case BackendGemini:
		return newGeminiModel(ctx, cfg)
	case BackendVertexAI:
		return newVertexAIModel(ctx, cfg)
	case BackendOpenRouter:
		return newOpenRouterModel(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", cfg.Backend)
	}
}

// newGeminiModel creates a Gemini model using Google AI Studio
func newGeminiModel(ctx context.Context, cfg Config) (model.LLM, error) {
	apiKey := cfg.GoogleAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Google API key is required for Gemini backend")
	}

	log.Printf("[Provider] Creating Gemini model: %s (Google AI Studio)", cfg.Model)

	clientConfig := &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	}

	return gemini.NewModel(ctx, cfg.Model, clientConfig)
}

// newVertexAIModel creates a Gemini model using Vertex AI
func newVertexAIModel(ctx context.Context, cfg Config) (model.LLM, error) {
	project := cfg.GCPProject
	if project == "" {
		project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if project == "" {
		return nil, fmt.Errorf("GCP Project is required for Vertex AI backend")
	}

	location := cfg.GCPLocation
	if location == "" {
		location = os.Getenv("GOOGLE_CLOUD_LOCATION")
	}
	if location == "" {
		return nil, fmt.Errorf("GCP Location is required for Vertex AI backend")
	}

	log.Printf("[Provider] Creating Gemini model: %s (Vertex AI, project: %s, location: %s)",
		cfg.Model, project, location)

	clientConfig := &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	}

	return gemini.NewModel(ctx, cfg.Model, clientConfig)
}

// newOpenRouterModel creates an OpenRouter model
func newOpenRouterModel(ctx context.Context, cfg Config) (model.LLM, error) {
	apiKey := cfg.OpenRouterAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required for OpenRouter backend")
	}

	log.Printf("[Provider] Creating OpenRouter model: %s", cfg.Model)

	orConfig := &openrouter.Config{
		APIKey:   apiKey,
		BaseURL:  cfg.OpenRouterBaseURL,
		SiteURL:  cfg.OpenRouterSiteURL,
		SiteName: cfg.OpenRouterSiteName,
	}

	return openrouter.NewModel(ctx, cfg.Model, orConfig)
}

// DetectBackend determines the backend to use based on configuration
func DetectBackend(useOpenRouter, useVertexAI bool) Backend {
	if useOpenRouter {
		return BackendOpenRouter
	}
	if useVertexAI {
		return BackendVertexAI
	}
	return BackendGemini
}

// DefaultModel returns the default model for each backend
func DefaultModel(backend Backend) string {
	switch backend {
	case BackendOpenRouter:
		return "google/gemini-2.5-flash" // Fast and cost-effective
	case BackendVertexAI, BackendGemini:
		return "gemini-2.5-flash"
	default:
		return "gemini-2.5-flash"
	}
}

// DefaultFallbackModel returns the default fallback model for each backend
func DefaultFallbackModel(backend Backend) string {
	switch backend {
	case BackendOpenRouter:
		return "google/gemini-3-pro-preview" // Higher quality fallback
	case BackendVertexAI, BackendGemini:
		return "gemini-2.5-pro"
	default:
		return "gemini-2.5-pro"
	}
}
