package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"webstar/noturno-leadgen-worker/internal/model/provider"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const (
	// DefaultExtractionTimeout is the timeout for extracting data from a single result
	DefaultExtractionTimeout = 30 * time.Second
	// MaxConcurrentExtractions limits how many extractions we run in parallel
	MaxConcurrentExtractions = 5
	// DefaultExtractorModel is the default Gemini model for data extraction
	DefaultExtractorModel = "gemini-2.5-flash"
	// DefaultExtractorFallbackModel is the fallback model when primary model quota is exceeded
	DefaultExtractorFallbackModel = "gemini-2.5-pro"
)

// ExtractedData contains structured company information extracted from scraped content
// @Description Company data extracted from website content
type ExtractedData struct {
	// URL of the source website
	URL string `json:"url"`
	// Company name extracted from the website
	Company string `json:"company,omitempty"`
	// Contact person name
	Contact string `json:"contact,omitempty"`
	// Contact person role/position
	ContactRole string `json:"contact_role,omitempty"`
	// Email addresses found (primary first)
	Emails []string `json:"emails,omitempty"`
	// Phone numbers found (primary first)
	Phones []string `json:"phones,omitempty"`
	// Physical address if available
	Address string `json:"address,omitempty"`
	// Website (canonical URL)
	Website string `json:"website,omitempty"`
	// Social media links
	SocialMedia map[string]string `json:"social_media,omitempty"`
	// Success indicates whether extraction was successful
	Success bool `json:"success"`
	// Error contains error message if extraction failed
	Error string `json:"error,omitempty"`
	// ExtractedAt timestamp
	ExtractedAt time.Time `json:"extracted_at"`
}

// DataExtractorConfig holds configuration for the DataExtractorHandler
type DataExtractorConfig struct {
	// APIKey is the Google API key for Gemini (used with Google AI Studio backend)
	APIKey string
	// Model is the Gemini model to use (default: gemini-2.5-flash for speed)
	Model string
	// FallbackModel is used when primary model quota is exceeded (default: gemini-2.5-pro)
	FallbackModel string
	// Timeout for extracting data from each result
	Timeout time.Duration
	// MaxConcurrent limits parallel extractions
	MaxConcurrent int
	// UseVertexAI enables Vertex AI backend instead of Google AI Studio
	UseVertexAI bool
	// GCPProject is the Google Cloud project ID (for Vertex AI backend)
	GCPProject string
	// GCPLocation is the Google Cloud location/region (for Vertex AI backend)
	GCPLocation string
	// UseOpenRouter enables OpenRouter backend instead of Google AI
	UseOpenRouter bool
	// OpenRouterAPIKey is the OpenRouter API key
	OpenRouterAPIKey string
	// OpenRouterBaseURL is the custom OpenRouter base URL (optional)
	OpenRouterBaseURL string
}

// DataExtractorHandler handles extracting structured data from scraped content using AI
type DataExtractorHandler struct {
	config         DataExtractorConfig
	agent          agent.Agent
	runner         *runner.Runner
	sessionService session.Service
	// Fallback resources
	fallbackAgent  agent.Agent
	fallbackRunner *runner.Runner
	clientConfig   *genai.ClientConfig
	// Provider backend
	backend       provider.Backend
	primaryModel  adkmodel.LLM
	fallbackModel adkmodel.LLM
	// Usage tracking
	usageTracker *UsageTrackerHandler
	// Current context for tracking
	currentUserID string
	currentJobID  *string
}

// NewDataExtractorHandler creates a new DataExtractorHandler instance
func NewDataExtractorHandler(config DataExtractorConfig) (*DataExtractorHandler, error) {
	// Check for OpenRouter configuration from env vars
	if os.Getenv("USE_OPENROUTER") == "true" {
		config.UseOpenRouter = true
	}
	if config.OpenRouterAPIKey == "" {
		config.OpenRouterAPIKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if config.OpenRouterBaseURL == "" {
		config.OpenRouterBaseURL = os.Getenv("OPENROUTER_BASE_URL")
	}

	// Check for Vertex AI configuration from env vars
	if os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "true" {
		config.UseVertexAI = true
	}
	if config.GCPProject == "" {
		config.GCPProject = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if config.GCPLocation == "" {
		config.GCPLocation = os.Getenv("GOOGLE_CLOUD_LOCATION")
	}

	// Determine backend
	backend := provider.DetectBackend(config.UseOpenRouter, config.UseVertexAI)

	// Validate configuration based on backend
	switch backend {
	case provider.BackendOpenRouter:
		if config.OpenRouterAPIKey == "" {
			return nil, fmt.Errorf("OpenRouter API key is required (set OPENROUTER_API_KEY env var or provide in config)")
		}
	case provider.BackendVertexAI:
		if config.GCPProject == "" {
			return nil, fmt.Errorf("GCP Project is required for Vertex AI (set GOOGLE_CLOUD_PROJECT env var)")
		}
		if config.GCPLocation == "" {
			return nil, fmt.Errorf("GCP Location is required for Vertex AI (set GOOGLE_CLOUD_LOCATION env var)")
		}
	default: // BackendGemini
		if config.APIKey == "" {
			config.APIKey = os.Getenv("GOOGLE_API_KEY")
		}
		if config.APIKey == "" {
			return nil, fmt.Errorf("Google API key is required (set GOOGLE_API_KEY env var)")
		}
	}

	// Set default model based on backend
	if config.Model == "" {
		if backend == provider.BackendOpenRouter {
			config.Model = os.Getenv("OPENROUTER_MODEL")
			if config.Model == "" {
				config.Model = provider.DefaultModel(backend)
			}
		} else {
			config.Model = os.Getenv("GEMINI_MODEL")
			if config.Model == "" {
				config.Model = DefaultExtractorModel
			}
		}
	}
	if config.FallbackModel == "" {
		config.FallbackModel = provider.DefaultFallbackModel(backend)
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultExtractionTimeout
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = MaxConcurrentExtractions
	}

	ctx := context.Background()

	// Create model using provider abstraction
	var llm adkmodel.LLM
	var clientConfig *genai.ClientConfig
	var err error

	if backend == provider.BackendOpenRouter {
		log.Printf("[DataExtractorHandler] Initializing with OpenRouter backend (model: %s)", config.Model)
		llm, err = provider.NewModel(ctx, provider.Config{
			Backend:           backend,
			Model:             config.Model,
			OpenRouterAPIKey:  config.OpenRouterAPIKey,
			OpenRouterBaseURL: config.OpenRouterBaseURL,
		})
	} else if backend == provider.BackendVertexAI {
		log.Printf("[DataExtractorHandler] Initializing with Vertex AI backend (project: %s, location: %s, model: %s)",
			config.GCPProject, config.GCPLocation, config.Model)
		clientConfig = &genai.ClientConfig{
			Project:  config.GCPProject,
			Location: config.GCPLocation,
			Backend:  genai.BackendVertexAI,
		}
		llm, err = gemini.NewModel(ctx, config.Model, clientConfig)
	} else {
		log.Printf("[DataExtractorHandler] Initializing with Google AI Studio backend (model: %s)", config.Model)
		clientConfig = &genai.ClientConfig{
			APIKey:  config.APIKey,
			Backend: genai.BackendGeminiAPI,
		}
		llm, err = gemini.NewModel(ctx, config.Model, clientConfig)
	}

	if err != nil {
		log.Printf("[DataExtractorHandler] Failed to create model: %v", err)
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Build instruction for the agent
	instruction := buildExtractorInstruction()

	// Create LLM agent for data extraction
	extractorAgent, err := llmagent.New(llmagent.Config{
		Name:        "data_extractor_agent",
		Model:       llm,
		Description: "An AI agent that extracts structured company contact information from website content.",
		Instruction: instruction,
	})
	if err != nil {
		log.Printf("[DataExtractorHandler] Failed to create agent: %v", err)
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Create session service and runner
	sessionService := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        "data_extractor",
		Agent:          extractorAgent,
		SessionService: sessionService,
	})
	if err != nil {
		log.Printf("[DataExtractorHandler] Failed to create runner: %v", err)
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	log.Printf("[DataExtractorHandler] Successfully initialized with model: %s (fallback: %s, backend: %s)",
		config.Model, config.FallbackModel, backend)

	return &DataExtractorHandler{
		config:         config,
		agent:          extractorAgent,
		runner:         r,
		sessionService: sessionService,
		clientConfig:   clientConfig,
		backend:        backend,
		primaryModel:   llm,
	}, nil
}

// initFallbackAgent initializes the fallback agent lazily when needed
func (h *DataExtractorHandler) initFallbackAgent() error {
	if h.fallbackRunner != nil {
		return nil // Already initialized
	}

	log.Printf("[DataExtractorHandler] Initializing fallback model: %s", h.config.FallbackModel)

	ctx := context.Background()

	// Create fallback model based on backend
	var fallbackLLM adkmodel.LLM
	var err error

	if h.backend == provider.BackendOpenRouter {
		fallbackLLM, err = provider.NewModel(ctx, provider.Config{
			Backend:           h.backend,
			Model:             h.config.FallbackModel,
			OpenRouterAPIKey:  h.config.OpenRouterAPIKey,
			OpenRouterBaseURL: h.config.OpenRouterBaseURL,
		})
	} else {
		fallbackLLM, err = gemini.NewModel(ctx, h.config.FallbackModel, h.clientConfig)
	}
	if err != nil {
		return fmt.Errorf("failed to create fallback model: %w", err)
	}

	h.fallbackModel = fallbackLLM

	// Build instruction for the agent
	instruction := buildExtractorInstruction()

	// Create fallback agent
	h.fallbackAgent, err = llmagent.New(llmagent.Config{
		Name:        "data_extractor_agent_fallback",
		Model:       fallbackLLM,
		Description: "An AI agent that extracts structured company contact information from website content (fallback).",
		Instruction: instruction,
	})
	if err != nil {
		return fmt.Errorf("failed to create fallback agent: %w", err)
	}

	// Create fallback runner
	h.fallbackRunner, err = runner.New(runner.Config{
		AppName:        "data_extractor_fallback",
		Agent:          h.fallbackAgent,
		SessionService: h.sessionService,
	})
	if err != nil {
		return fmt.Errorf("failed to create fallback runner: %w", err)
	}

	log.Printf("[DataExtractorHandler] Fallback model initialized successfully: %s", h.config.FallbackModel)
	return nil
}

// SetUsageTracker sets the usage tracker for recording AI usage metrics
func (h *DataExtractorHandler) SetUsageTracker(tracker *UsageTrackerHandler) {
	h.usageTracker = tracker
}

// SetUserContext sets the current user and job context for usage tracking
func (h *DataExtractorHandler) SetUserContext(userID string, jobID *string) {
	h.currentUserID = userID
	h.currentJobID = jobID
}

// ClearUserContext clears the user context after processing
func (h *DataExtractorHandler) ClearUserContext() {
	h.currentUserID = ""
	h.currentJobID = nil
}

// isExtractorQuotaExceededError checks if the error is a quota exceeded (429) error
func isExtractorQuotaExceededError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "429") || strings.Contains(errStr, "RESOURCE_EXHAUSTED") || strings.Contains(errStr, "quota")
}

// buildExtractorInstruction creates the instruction prompt for the data extractor agent
func buildExtractorInstruction() string {
	return `You are a data extraction specialist. Your task is to extract structured contact information from website content.

Given website content in markdown format, extract the following information:

1. **Company**: The official company/business name
2. **Contact**: Name of a contact person (preferably the owner, manager, or key decision maker)
3. **ContactRole**: The role/position of the contact person (e.g., "CEO", "Diretor", "Gerente")
4. **Emails**: All email addresses found (list the primary/contact email first)
5. **Phones**: All phone numbers found (list the primary/contact number first)
6. **Address**: Physical address if available
7. **Website**: The canonical website URL
8. **SocialMedia**: Social media profile URLs (LinkedIn, Facebook, Instagram, Twitter, etc.)

IMPORTANT RULES:
- Extract ONLY information that is explicitly present in the content
- Do NOT invent or guess information
- If information is not found, leave the field empty
- For emails and phones, extract ALL that you find
- Prefer Brazilian formats for phones (e.g., +55 81 99999-9999)
- Clean and normalize phone numbers (remove extra spaces, standardize format)
- For social media, extract the full URL

OUTPUT FORMAT:
You MUST respond with ONLY a valid JSON object in this exact format (no markdown, no code blocks, no explanations):
{
  "company": "Company Name",
  "contact": "Contact Person Name",
  "contact_role": "Role/Position",
  "emails": ["email1@example.com", "email2@example.com"],
  "phones": ["+55 11 99999-9999", "+55 11 3333-3333"],
  "address": "Full address if available",
  "website": "https://www.example.com",
  "social_media": {
    "linkedin": "https://linkedin.com/company/example",
    "instagram": "https://instagram.com/example"
  }
}

If no information can be extracted, respond with:
{"company": "", "contact": "", "contact_role": "", "emails": [], "phones": [], "address": "", "website": "", "social_media": {}}`
}

// ExtractData extracts structured data from a single organic result
func (h *DataExtractorHandler) ExtractData(ctx context.Context, result OrganicResult) *ExtractedData {
	startTime := time.Now()
	extracted := &ExtractedData{
		URL:         result.Link,
		Website:     result.Link,
		ExtractedAt: time.Now(),
	}

	// Skip if no scraped content
	if result.ScrapedContent == "" {
		extracted.Error = "no scraped content available"
		extracted.Success = false
		return extracted
	}

	// Build prompt
	prompt := h.buildPrompt(result)
	modelUsed := h.config.Model

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Create user message
	userMessage := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: prompt},
		},
	}

	// Create session for this extraction
	userID := "system"
	createResp, err := h.sessionService.Create(ctx, &session.CreateRequest{
		AppName: "data_extractor",
		UserID:  userID,
	})
	if err != nil {
		log.Printf("[DataExtractorHandler] Failed to create session for %s: %v", result.Link, err)
		extracted.Error = fmt.Sprintf("failed to create session: %v", err)
		extracted.Success = false
		return extracted
	}
	sessionID := createResp.Session.ID()
	defer func() {
		// Clean up session after use
		_ = h.sessionService.Delete(ctx, &session.DeleteRequest{
			AppName:   "data_extractor",
			UserID:    userID,
			SessionID: sessionID,
		})
	}()

	// Run the agent
	var responseText string
	var extractionErr error
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	}

	log.Printf("[DataExtractorHandler] Extracting data for: %s (session: %s)", result.Link, sessionID)

	// Try with primary model
	for event, err := range h.runner.Run(ctx, userID, sessionID, userMessage, runConfig) {
		if err != nil {
			extractionErr = err
			break
		}

		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					responseText += part.Text
				}
			}
		}
	}

	// If primary model failed with quota error, try fallback
	if extractionErr != nil && isExtractorQuotaExceededError(extractionErr) {
		log.Printf("[DataExtractorHandler] Quota exceeded for primary model, trying fallback: %s", h.config.FallbackModel)

		// Initialize fallback agent if needed
		if err := h.initFallbackAgent(); err != nil {
			log.Printf("[DataExtractorHandler] Failed to initialize fallback agent: %v", err)
			extracted.Error = fmt.Sprintf("extraction failed (primary quota exceeded, fallback init failed): %v", err)
			extracted.Success = false
			return extracted
		}

		// Create a new session for fallback
		fallbackResp, err := h.sessionService.Create(ctx, &session.CreateRequest{
			AppName: "data_extractor_fallback",
			UserID:  userID,
		})
		if err != nil {
			log.Printf("[DataExtractorHandler] Failed to create fallback session: %v", err)
			extracted.Error = fmt.Sprintf("extraction failed (fallback session error): %v", err)
			extracted.Success = false
			return extracted
		}
		fallbackSessionID := fallbackResp.Session.ID()
		defer func() {
			_ = h.sessionService.Delete(ctx, &session.DeleteRequest{
				AppName:   "data_extractor_fallback",
				UserID:    userID,
				SessionID: fallbackSessionID,
			})
		}()

		// Reset for fallback attempt
		responseText = ""
		extractionErr = nil

		log.Printf("[DataExtractorHandler] Retrying with fallback model for: %s (session: %s)", result.Link, fallbackSessionID)
		modelUsed = h.config.FallbackModel

		for event, err := range h.fallbackRunner.Run(ctx, userID, fallbackSessionID, userMessage, runConfig) {
			if err != nil {
				extractionErr = err
				break
			}

			if event.Content != nil {
				for _, part := range event.Content.Parts {
					if part.Text != "" {
						responseText += part.Text
					}
				}
			}
		}
	}

	// Handle final error
	if extractionErr != nil {
		log.Printf("[DataExtractorHandler] Error during extraction for %s: %v", result.Link, extractionErr)
		extracted.Error = fmt.Sprintf("extraction failed: %v", extractionErr)
		extracted.Success = false
		// Track failed extraction
		if h.usageTracker != nil {
			errMsg := extractionErr.Error()
			h.usageTracker.TrackDataExtraction(h.currentUserID, h.currentJobID, nil, modelUsed, prompt, "", startTime, false, &errMsg)
		}
		return extracted
	}

	// Parse the response
	h.parseResponse(responseText, extracted)

	// If we couldn't extract company name from AI, try to get it from the title
	if extracted.Company == "" && result.Title != "" {
		extracted.Company = result.Title
	}

	// Try to extract emails/phones from content if AI didn't find them
	if len(extracted.Emails) == 0 {
		extracted.Emails = extractEmailsFromText(result.ScrapedContent)
	}
	if len(extracted.Phones) == 0 {
		extracted.Phones = extractPhonesFromText(result.ScrapedContent)
	}

	extracted.Success = true

	// Track successful extraction
	if h.usageTracker != nil {
		h.usageTracker.TrackDataExtraction(h.currentUserID, h.currentJobID, nil, modelUsed, prompt, responseText, startTime, true, nil)
	}

	return extracted
}

// buildPrompt creates the extraction prompt for a single result
func (h *DataExtractorHandler) buildPrompt(result OrganicResult) string {
	// Limit content length to avoid token limits
	content := result.ScrapedContent
	maxLen := 15000 // ~3750 tokens
	if len(content) > maxLen {
		content = content[:maxLen] + "\n\n[Content truncated...]"
	}

	return fmt.Sprintf(`Extract contact information from the following website content.

Website URL: %s
Website Title: %s

---
CONTENT:
%s
---

Extract all contact information and respond with ONLY a JSON object.`, result.Link, result.Title, content)
}

// parseResponse parses the AI response into ExtractedData
func (h *DataExtractorHandler) parseResponse(response string, data *ExtractedData) {
	// Clean response - remove markdown code blocks if present
	response = cleanJSONResponse(response)

	// Try to find JSON in the response
	start := findChar(response, '{')
	end := findLastChar(response, '}')

	if start == -1 || end == -1 || end <= start {
		log.Printf("[DataExtractorHandler] No valid JSON found in response")
		return
	}

	jsonStr := response[start : end+1]

	// Parse using simple extraction (avoid importing encoding/json to keep it light)
	data.Company = extractJSONString(jsonStr, "company")
	data.Contact = extractJSONString(jsonStr, "contact")
	data.ContactRole = extractJSONString(jsonStr, "contact_role")
	data.Address = extractJSONString(jsonStr, "address")
	data.Website = extractJSONString(jsonStr, "website")
	data.Emails = extractJSONArray(jsonStr, "emails")
	data.Phones = extractJSONArray(jsonStr, "phones")
	data.SocialMedia = extractJSONObject(jsonStr, "social_media")
}

// ExtractFromResults extracts data from multiple organic results concurrently
func (h *DataExtractorHandler) ExtractFromResults(ctx context.Context, results []OrganicResult) map[string]*ExtractedData {
	extractedMap := make(map[string]*ExtractedData)
	if len(results) == 0 {
		return extractedMap
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, h.config.MaxConcurrent)

	for i := range results {
		result := results[i]
		if result.ScrapedContent == "" {
			continue // Skip results without scraped content
		}

		wg.Add(1)
		go func(r OrganicResult) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Printf("[DataExtractorHandler] Extracting data from: %s", r.Link)
			extracted := h.ExtractData(ctx, r)

			mu.Lock()
			extractedMap[r.Link] = extracted
			mu.Unlock()

			if extracted.Success {
				log.Printf("[DataExtractorHandler] Successfully extracted data from: %s (Company: %s, Emails: %d, Phones: %d)",
					r.Link, extracted.Company, len(extracted.Emails), len(extracted.Phones))
			} else {
				log.Printf("[DataExtractorHandler] Failed to extract from %s: %s", r.Link, extracted.Error)
			}
		}(result)
	}

	wg.Wait()
	return extractedMap
}

// Helper functions

func cleanJSONResponse(response string) string {
	// Remove markdown code blocks
	response = removePrefix(response, "```json")
	response = removePrefix(response, "```")
	response = removeSuffix(response, "```")
	return trimValue(response)
}

func removePrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func removeSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func findChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func findLastChar(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// extractJSONString extracts a string value from a JSON-like string
func extractJSONString(json, key string) string {
	// Find "key": "value" pattern
	keyPattern := fmt.Sprintf(`"%s"\s*:\s*"`, key)
	re := regexp.MustCompile(keyPattern)
	loc := re.FindStringIndex(json)
	if loc == nil {
		return ""
	}

	start := loc[1]
	// Find the closing quote, handling escaped quotes
	inEscape := false
	for i := start; i < len(json); i++ {
		if inEscape {
			inEscape = false
			continue
		}
		if json[i] == '\\' {
			inEscape = true
			continue
		}
		if json[i] == '"' {
			return json[start:i]
		}
	}
	return ""
}

// extractJSONArray extracts an array of strings from a JSON-like string
func extractJSONArray(json, key string) []string {
	var result []string

	// Find "key": [ pattern
	keyPattern := fmt.Sprintf(`"%s"\s*:\s*\[`, key)
	re := regexp.MustCompile(keyPattern)
	loc := re.FindStringIndex(json)
	if loc == nil {
		return result
	}

	start := loc[1]
	// Find the closing bracket
	depth := 1
	end := start
	for i := start; i < len(json) && depth > 0; i++ {
		if json[i] == '[' {
			depth++
		} else if json[i] == ']' {
			depth--
			if depth == 0 {
				end = i
			}
		}
	}

	if end <= start {
		return result
	}

	arrayContent := json[start:end]

	// Extract quoted strings from array
	stringRe := regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
	matches := stringRe.FindAllStringSubmatch(arrayContent, -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			result = append(result, match[1])
		}
	}

	return result
}

// extractJSONObject extracts a map from a JSON-like string
func extractJSONObject(json, key string) map[string]string {
	result := make(map[string]string)

	// Find "key": { pattern
	keyPattern := fmt.Sprintf(`"%s"\s*:\s*\{`, key)
	re := regexp.MustCompile(keyPattern)
	loc := re.FindStringIndex(json)
	if loc == nil {
		return result
	}

	start := loc[1]
	// Find the closing brace
	depth := 1
	end := start
	for i := start; i < len(json) && depth > 0; i++ {
		if json[i] == '{' {
			depth++
		} else if json[i] == '}' {
			depth--
			if depth == 0 {
				end = i
			}
		}
	}

	if end <= start {
		return result
	}

	objContent := json[start:end]

	// Extract key-value pairs
	pairRe := regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]*)"`)
	matches := pairRe.FindAllStringSubmatch(objContent, -1)
	for _, match := range matches {
		if len(match) > 2 {
			result[match[1]] = match[2]
		}
	}

	return result
}

// extractEmailsFromText extracts email addresses from text using regex
func extractEmailsFromText(text string) []string {
	emailRe := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	matches := emailRe.FindAllString(text, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, email := range matches {
		lower := toLowerASCII(email)
		if !seen[lower] {
			seen[lower] = true
			unique = append(unique, email)
		}
	}
	return unique
}

// extractPhonesFromText extracts phone numbers from text using regex
func extractPhonesFromText(text string) []string {
	// Match various phone formats including Brazilian
	phoneRe := regexp.MustCompile(`(?:\+55\s?)?(?:\(?\d{2}\)?\s?)?(?:9?\d{4}[-.\s]?\d{4})`)
	matches := phoneRe.FindAllString(text, -1)

	// Deduplicate and clean
	seen := make(map[string]bool)
	var unique []string
	for _, phone := range matches {
		cleaned := cleanPhone(phone)
		if len(cleaned) >= 8 && !seen[cleaned] {
			seen[cleaned] = true
			unique = append(unique, phone)
		}
	}
	return unique
}

func cleanPhone(phone string) string {
	var result []byte
	for i := 0; i < len(phone); i++ {
		if phone[i] >= '0' && phone[i] <= '9' {
			result = append(result, phone[i])
		}
	}
	return string(result)
}

func toLowerASCII(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
