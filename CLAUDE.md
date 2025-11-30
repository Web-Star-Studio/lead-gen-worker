# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based REST API service for scraping Google search results via SerpAPI with automatic website content extraction via Firecrawl. It's designed for lead generation workflows, extracting organic search results with domain exclusion, pagination, and optional website scraping capabilities.

**Tech Stack:** Go 1.25.4, Gin web framework, SerpAPI, Firecrawl, Swagger (OpenAPI 3.1)

## Common Commands

### Development
```bash
# Run in development mode (requires SERPAPI_KEY env var)
export SERPAPI_KEY="your-api-key"
export FIRECRAWL_API_KEY="your-firecrawl-key"  # Optional: enables website scraping
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
      └── firecrawlHandler.go      # Website scraping via Firecrawl API
docs/                        # Auto-generated Swagger documentation
```

### Request Flow

1. **HTTP Request** → `router.go` (Gin router with Logger & Recovery middleware)
2. **Controller Layer** → `controllers/search_controller.go` (validates request, converts DTO)
3. **Business Logic** → `handlers/googleSearchHandler.go` (SerpAPI integration)
4. **External API** → SerpAPI (location resolution + Google Light search engine)
5. **Website Scraping** → `handlers/firecrawlHandler.go` (if configured, scrapes each result's homepage)
6. **Response** → Enriched organic results with scraped markdown content + pagination metadata

### Key Design Patterns

**Layered Architecture:**
- **Controllers**: Handle HTTP concerns (request binding, response formatting, status codes)
- **Handlers**: Contain business logic (SerpAPI calls, location resolution, query building)
- **DTOs**: Separate external API contracts from internal data structures

**Location Resolution**: The service automatically resolves location names (e.g., "Recife") to canonical forms (e.g., "Recife,State of Pernambuco,Brazil") via SerpAPI's locations endpoint before searching.

**Domain Exclusion**: Implements `-site:` Google operator by appending excluded domains to the query string in `googleSearchHandler.go:132`.

**Type Safety**: Uses helper functions (`getString`, `getInt`, `getFloat`) in `googleSearchHandler.go:217-236` to safely extract values from SerpAPI's `map[string]interface{}` responses.

**Automatic Website Scraping**: When `FIRECRAWL_API_KEY` is configured, the `FirecrawlHandler` is automatically attached to `GoogleSearchHandler`. After fetching search results, each organic result's website is scraped concurrently (up to 5 parallel requests) and the markdown content is added to the response.

## Configuration

**Required:**
- `SERPAPI_KEY` - Your SerpAPI API key (fatal error if missing)

**Optional:**
- `PORT` - HTTP server port (default: 8080)
- `GIN_MODE` - Set to "release" for production (hides debug routes)
- `FIRECRAWL_API_KEY` - Your Firecrawl API key (enables automatic website scraping)
- `FIRECRAWL_API_URL` - Custom Firecrawl API URL (leave empty for default cloud API)

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

## Type Declaration Rule

NEVER use `any` for type declarations. Always create explicit type interfaces. See existing patterns in:
- `handlers/googleSearchHandler.go` - Defines explicit structs for all SerpAPI response types
- `handlers/firecrawlHandler.go` - Defines `ScrapedPage` struct for Firecrawl responses
- `dto/search.go` - Defines request/response contracts with proper types

## Response Fields (OrganicResult)

When Firecrawl is enabled, each `OrganicResult` includes:
- `scraped_content` (string, optional) - Markdown content from the website homepage
- `scrape_error` (string, optional) - Error message if scraping failed

Example response with scraped content:
```json
{
  "position": 1,
  "title": "Example Company",
  "link": "https://example.com",
  "snippet": "...",
  "scraped_content": "# Welcome to Example\n\nWe provide...",
  "scrape_error": ""
}
```
