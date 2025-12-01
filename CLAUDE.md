# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based REST API service for scraping Google search results via SerpAPI with automatic website content extraction via Firecrawl. It's designed for lead generation workflows, extracting organic search results with domain exclusion, pagination, and optional website scraping capabilities.

**Tech Stack:** Go 1.25.4, Gin web framework, SerpAPI, Firecrawl, Supabase, Swagger (OpenAPI 3.1)

## Common Commands

### Development
```bash
# Run in development mode (requires SERPAPI_KEY env var)
export SERPAPI_KEY="your-api-key"
export FIRECRAWL_API_KEY="your-firecrawl-key"  # Optional: enables website scraping
export SUPABASE_URL="https://xxx.supabase.co"  # Optional: enables database access
export SUPABASE_SECRET_KEY="sb_secret_xxx"    # Optional: Supabase secret key (bypasses RLS)
go run ./cmd/api

# Build the binary
go build -o bin/api ./cmd/api

# Run the compiled binary
./bin/api
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Run tests for a specific package
go test ./internal/api/controllers -v

# Run a specific test function
go test ./internal/api/controllers -run TestSearch_Success -v

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Swagger Documentation

```bash
# Install swag v2 CLI tool
go install github.com/swaggo/swag/v2/cmd/swag@latest

# Regenerate Swagger docs (after changing API annotations)
swag init -g cmd/api/main.go -o docs --v3.1
```

### Dependencies

```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

## Architecture

### Directory Structure

```
cmd/api/                     # Application entry point
internal/                    # Private application code
  ├── api/                   # HTTP layer
  │   ├── router.go          # Gin router setup, middleware, route registration
  │   └── controllers/       # HTTP request handlers (thin layer)
  ├── config/                # Environment variable configuration
  ├── dto/                   # Data Transfer Objects (request/response schemas)
  └── handlers/              # Business logic layer
      ├── googleSearchHandler.go   # SerpAPI integration, search orchestration
      ├── firecrawlHandler.go      # Website scraping via Firecrawl API
      ├── dataExtractorHandler.go  # AI-powered data extraction (company info, contacts)
      ├── preCallReportHandler.go  # AI-powered pre-call reports via Google ADK
      └── supabaseHandler.go       # Database operations via Supabase
docs/                        # Auto-generated Swagger documentation
```

### Request Flow

1. **HTTP Request** → `router.go` (Gin router with Logger & Recovery middleware)
2. **Controller Layer** → `controllers/search_controller.go` (validates request, converts DTO)
3. **Business Logic** → `handlers/googleSearchHandler.go` (SerpAPI integration)
4. **External API** → SerpAPI (location resolution + Google Light search engine)
5. **Website Scraping** → `handlers/firecrawlHandler.go` (if configured, scrapes each result's homepage)
6. **Data Extraction** → `handlers/dataExtractorHandler.go` (AI extracts company data from scraped content)
7. **Pre-Call Reports** → `handlers/preCallReportHandler.go` (AI generates sales reports)
8. **Response** → Enriched organic results with scraped content, extracted data, and pre-call reports

### Key Design Patterns

**Layered Architecture:**
- **Controllers**: Handle HTTP concerns (request binding, response formatting, status codes)
- **Handlers**: Contain business logic (SerpAPI calls, location resolution, query building)
- **DTOs**: Separate external API contracts from internal data structures

**Location Resolution**: The service automatically resolves location names (e.g., "Recife") to canonical forms (e.g., "Recife,State of Pernambuco,Brazil") via SerpAPI's locations endpoint before searching.

**Domain Exclusion**: Implements `-site:` Google operator by appending excluded domains to the query string in `googleSearchHandler.go:132`.

**Type Safety**: Uses helper functions (`getString`, `getInt`, `getFloat`) in `googleSearchHandler.go:217-236` to safely extract values from SerpAPI's `map[string]interface{}` responses.

**Automatic Website Scraping**: When `FIRECRAWL_API_KEY` is configured, the `FirecrawlHandler` is automatically attached to `GoogleSearchHandler`. After fetching search results, each organic result's website is scraped concurrently (up to 5 parallel requests) and the markdown content is added to the response.

**AI Pre-Call Reports**: When `GOOGLE_API_KEY` is configured, the `PreCallReportHandler` uses Google ADK with Gemini to automatically generate comprehensive pre-call reports for each search result. Reports include company summary, key services, talking points, pain points, and recommended approach.

## Configuration

**Required:**

- `SERPAPI_KEY` - Your SerpAPI API key (fatal error if missing)

**Optional:**

- `PORT` - HTTP server port (default: 8080)
- `GIN_MODE` - Set to "release" for production (hides debug routes)
- `FIRECRAWL_API_KEY` - Your Firecrawl API key (enables automatic website scraping)
- `FIRECRAWL_API_URL` - Custom Firecrawl API URL (leave empty for default cloud API)
- `SUPABASE_URL` - Your Supabase project URL (e.g., "https://xxx.supabase.co")
- `SUPABASE_SECRET_KEY` - Your Supabase secret key (`sb_secret_xxx` format) - bypasses RLS for server-side operations. Get it from Dashboard > Settings > API Keys > Secret Key. (Falls back to legacy `SUPABASE_KEY` for backward compatibility)

**AI Features (Data Extraction + Pre-Call Reports):**

Choose one backend:

**Google AI Studio** (simpler, uses API key):

- `GOOGLE_API_KEY` - Your Google API key for Gemini
- `GEMINI_MODEL` - Gemini model for reports (default: gemini-2.5-pro-preview-06-05)

**Vertex AI** (enterprise, uses GCP project):

- `GOOGLE_GENAI_USE_VERTEXAI=true` - Enable Vertex AI backend
- `GOOGLE_CLOUD_PROJECT` - Your GCP project ID
- `GOOGLE_CLOUD_LOCATION` - GCP region (e.g., "us-central1")
- `GEMINI_MODEL` - Gemini model (default: gemini-2.5-pro-preview-06-05)

## API Endpoints

- `GET /health` - Health check (returns `{"status": "ok"}`)
- `POST /api/v1/search` - Main search endpoint (see SearchRequest DTO)
- `GET /swagger/index.html` - Interactive API documentation

## Testing Strategy

The codebase uses `testify` for assertions (`assert` for soft failures, `require` for hard stops) and `httptest` for HTTP handler testing.

**Test Coverage Areas:**

- `internal/api/router_test.go` - Health check, route registration, 404 handling
- `internal/api/controllers/*_test.go` - Request validation, HTTP status codes, error responses
- `internal/dto/*_test.go` - JSON marshaling/unmarshaling, field validation
- `internal/handlers/*_test.go` - Business logic, SerpAPI response parsing

## Important Implementation Notes

- **No Global State**: Configuration is passed via dependency injection (see `cmd/api/main.go:31-51`)
- **Pagination**: `num` parameter is clamped (default: 10, max: 100) in `googleSearchHandler.go:137-142`
- **Swagger v2 with OpenAPI 3.1**: Uses `swaggo/swag/v2` for OpenAPI 3.1 spec generation (important for correct `servers` array and `requestBody` schemas)
- **Type Annotations**: All DTOs and response types use Swagger annotations for documentation generation
- **Firecrawl Integration**: Internal handler (not exposed as API) that automatically scrapes websites after search completes
- **Concurrent Scraping**: Up to `MaxConcurrentScrapes` (5) websites scraped in parallel with 30-second timeout per site
- **Graceful Degradation**: If a website fails to scrape, the result includes `scrape_error` but doesn't fail the entire request

## Supabase Integration

The `SupabaseHandler` provides database access via Supabase. Key methods:

- `NewSupabaseHandler(url, key)` - Initialize the handler
- `GetRows(table, columns)` - Retrieve all rows from a table
- `GetRowsWithFilter(table, columns, filterColumn, filterValue)` - Query with filtering
- `GetRowByID(table, columns, id)` - Get a single row by ID
- `GetClient()` - Access the underlying Supabase client for advanced operations

Example usage:
```go
handler, err := handlers.NewSupabaseHandler(url, key)
result, err := handler.GetRows("users", "id,name,email")
for _, row := range result.Data {
    fmt.Println(row["name"])
}
```

## Type Declaration Rule

NEVER use `any` for type declarations. Always create explicit type interfaces. See existing patterns in:

- `handlers/googleSearchHandler.go` - Defines explicit structs for all SerpAPI response types
- `handlers/firecrawlHandler.go` - Defines `ScrapedPage` struct for Firecrawl responses
- `handlers/preCallReportHandler.go` - Defines `PreCallReport` struct for AI-generated reports
- `handlers/supabaseHandler.go` - Defines `QueryResult` struct for database queries
- `dto/search.go` - Defines request/response contracts with proper types

## Response Fields (OrganicResult)

When AI features are enabled, each `OrganicResult` includes:

- `scraped_content` (string, optional) - Markdown content from the website homepage
- `scrape_error` (string, optional) - Error message if scraping failed
- `extracted_data` (object, optional) - Structured company data extracted by AI

Example response with all features:
```json
{
  "position": 1,
  "title": "Example Company",
  "link": "https://example.com",
  "snippet": "...",
  "scraped_content": "# Welcome to Example\n\nWe provide...",
  "extracted_data": {
    "url": "https://example.com",
    "company": "Example Corp",
    "contact": "John Doe",
    "contact_role": "CEO",
    "emails": ["contact@example.com"],
    "phones": ["+55 11 99999-9999"],
    "address": "123 Main St, São Paulo",
    "social_media": {
      "linkedin": "https://linkedin.com/company/example"
    },
    "success": true
  }
}
```

## Data Extractor (Google ADK)

The `DataExtractorHandler` is an AI agent that extracts structured company contact information from scraped website content.

### Architecture

- Uses Google ADK's `llmagent` with Gemini Flash model (faster extraction)
- Processes results concurrently (max 5 parallel extractions)
- 30-second timeout per extraction
- Falls back to regex extraction if AI fails

### ExtractedData Fields

Each extraction includes:

- `company` - Company/business name
- `contact` - Contact person name
- `contact_role` - Role/position (e.g., "CEO", "Diretor")
- `emails` - List of email addresses found
- `phones` - List of phone numbers found
- `address` - Physical address if available
- `website` - Canonical website URL
- `social_media` - Map of social media profiles (LinkedIn, Instagram, etc.)

### Example Usage

```go
handler, err := handlers.NewDataExtractorHandler(handlers.DataExtractorConfig{
    APIKey: os.Getenv("GOOGLE_API_KEY"),
    Model:  "gemini-2.5-flash", // Fast model for extraction
})

// Automatically integrated via GoogleSearchHandler
searchHandler.SetDataExtractorHandler(handler)
```

## Pre-Call Report Integration (Google ADK)

The `PreCallReportHandler` is an AI agent built with Google ADK that generates comprehensive pre-call reports for sales leads.

### Architecture

- Uses Google ADK's `llmagent` for LLM-based report generation
- Powered by Gemini model (configurable via `GEMINI_MODEL` env var)
- Processes results concurrently (max 3 parallel reports)
- 60-second timeout per report generation

### PreCallReport Fields

Each report includes:

- `company_name` - Extracted company name
- `industry` - Business sector
- `company_summary` - AI-generated overview
- `key_services` - List of main offerings
- `target_audience` - Primary customer segments
- `potential_pain_points` - Challenges to address in sales call
- `talking_points` - Suggested conversation starters
- `competitive_advantages` - Unique selling points
- `contact_info` - Extracted contact details
- `recommended_approach` - Sales strategy recommendation

### Example Usage

```go
handler, err := handlers.NewPreCallReportHandler(handlers.PreCallReportConfig{
    APIKey:  os.Getenv("GOOGLE_API_KEY"),
    Model:   "gemini-2.5-pro-preview-06-05",
    Timeout: 90 * time.Second,
})

// Automatically integrated via GoogleSearchHandler
searchHandler.SetPreCallReportHandler(handler)
```

### Response with Pre-Call Reports

When `GOOGLE_API_KEY` is configured, the search response includes:

```json
{
  "organic_results": [...],
  "pre_call_reports": {
    "https://example.com": {
      "url": "https://example.com",
      "company_name": "Example Corp",
      "industry": "Technology",
      "company_summary": "A leading software company...",
      "key_services": ["Web Development", "Cloud Solutions"],
      "talking_points": ["Discuss scaling challenges", "Demo our automation tools"],
      "success": true
    }
  }
}
```
