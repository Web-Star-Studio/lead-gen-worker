# Lead Gen Worker

A high-performance REST API service for scraping Google search results using [SerpAPI](https://serpapi.com/). Built with Go and the Gin web framework, this service is designed for lead generation workflows by extracting organic search results with advanced filtering capabilities.

## Features

- 🔍 **Google Search Integration** - Powered by SerpAPI's Google Light engine
- 🌍 **Location-aware searches** - Automatic canonical location resolution
- 🚫 **Domain exclusion** - Filter out unwanted domains (e.g., social media, directories)
- 📄 **Automatic multi-page fetching** - Request up to 100 results; the API automatically fetches multiple pages from SerpAPI
- ⚡ **High performance** - Built with Gin, one of the fastest Go web frameworks
- 🏗️ **Clean architecture** - Follows Go best practices for project structure
- 🧪 **Comprehensive testing** - Unit tests with testify for all components

## Project Structure

```
lead-gen-worker/
├── cmd/
│   └── api/
│       └── main.go                    # Application entry point
├── internal/
│   ├── api/
│   │   ├── router.go                  # Gin router configuration
│   │   └── controllers/
│   │       └── search_controller.go   # HTTP request handlers
│   ├── config/
│   │   └── config.go                  # Environment configuration
│   ├── dto/
│   │   └── search.go                  # Request/Response data structures
│   └── handlers/
│       └── googleSearchHandler.go     # SerpAPI business logic
├── docs/
│   ├── docs.go                        # Swagger embedded spec
│   ├── swagger.json                   # OpenAPI JSON spec
│   └── swagger.yaml                   # OpenAPI YAML spec
├── go.mod
├── go.sum
└── README.md
```

## Prerequisites

- **Go** 1.21 or higher
- **SerpAPI Account** - Get your API key at [serpapi.com](https://serpapi.com/)

## Installation

### Clone the repository

```bash
git clone https://github.com/your-username/lead-gen-worker.git
cd lead-gen-worker
```

### Install dependencies

```bash
go mod download
```

### Build the application

```bash
go build -o bin/api ./cmd/api
```

## Configuration

The application is configured via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SERPAPI_KEY` | ✅ Yes | - | Your SerpAPI API key |
| `PORT` | No | `8080` | HTTP server port |

### Setting environment variables

**Linux/macOS:**
```bash
export SERPAPI_KEY="your-api-key-here"
export PORT="8080"
```

**Windows (PowerShell):**
```powershell
$env:SERPAPI_KEY="your-api-key-here"
$env:PORT="8080"
```

**Windows (CMD):**
```cmd
set SERPAPI_KEY=your-api-key-here
set PORT=8080
```

## Running the Application

### Development mode

```bash
export SERPAPI_KEY="your-api-key-here"
go run ./cmd/api
```

### Production mode

```bash
export SERPAPI_KEY="your-api-key-here"
export GIN_MODE=release
./bin/api
```

You should see:
```
[GIN-debug] POST   /api/v1/search            --> webstar/noturno-leadgen-worker/internal/api/controllers.(*SearchController).Search-fm (3 handlers)
[GIN-debug] GET    /health                   --> webstar/noturno-leadgen-worker/internal/api.NewRouter.func1 (3 handlers)
[GIN-debug] GET    /swagger/*any             --> github.com/swaggo/gin-swagger.WrapHandler.func1 (3 handlers)
2024/01/15 10:30:00 Server starting on port 8080
2024/01/15 10:30:00 Swagger UI available at http://localhost:8080/swagger/index.html
```

## API Documentation

### Base URL

```
http://localhost:8080
```

### Swagger UI

Interactive API documentation is available at:

```
http://localhost:8080/swagger/index.html
```

The Swagger UI allows you to:
- Browse all available endpoints
- View request/response schemas
- Test API calls directly from the browser

---

### Health Check

Check if the service is running.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "ok"
}
```

**Example:**
```bash
curl http://localhost:8080/health
```

---

### Search

Perform a Google search and retrieve organic results.

**Endpoint:** `POST /api/v1/search`

**Content-Type:** `application/json`

#### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `q` | string | ✅ Yes | Search query |
| `location` | string | ✅ Yes | Location for the search (e.g., "Recife", "São Paulo") |
| `hl` | string | No | Language code (e.g., "pt-br", "en") |
| `gl` | string | No | Country code (e.g., "br", "us") |
| `exclude_domains` | string[] | No | List of domains to exclude from results |
| `num` | integer | No | Number of results (default: 10, max: 100) |
| `start` | integer | No | Result offset for pagination (default: 0) |

#### Response

| Field | Type | Description |
|-------|------|-------------|
| `organic_results` | array | List of organic search results |
| `organic_results[].position` | integer | Result position |
| `organic_results[].title` | string | Page title |
| `organic_results[].link` | string | Page URL |
| `organic_results[].displayed_link` | string | Displayed URL |
| `organic_results[].snippet` | string | Result snippet/description |
| `organic_results[].rating` | float | Rating (if available) |
| `organic_results[].reviews` | integer | Number of reviews (if available) |
| `organic_results[].sitelinks` | object | Sitelinks (if available) |
| `serpapi_pagination` | object | Pagination information |
| `serpapi_pagination.current` | integer | Current page number |
| `serpapi_pagination.next` | string | URL for next page of results |

#### Error Response

| Field | Type | Description |
|-------|------|-------------|
| `error` | string | Error message |

---

## Examples

### Basic Search

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "q": "escritório de contabilidade",
    "location": "Recife",
    "hl": "pt-br",
    "gl": "br"
  }'
```

### Search with Domain Exclusion

Exclude social media and directory sites from results:

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "q": "escritório de contabilidade",
    "location": "Recife",
    "hl": "pt-br",
    "gl": "br",
    "exclude_domains": [
      "instagram.com",
      "linkedin.com",
      "facebook.com",
      "twitter.com",
      "ohub.com.br",
      "yelp.com"
    ]
  }'
```

### Search with Pagination

Get 50 results starting from the first page:

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "q": "advogado trabalhista",
    "location": "São Paulo",
    "hl": "pt-br",
    "gl": "br",
    "num": 50,
    "start": 0
  }'
```

Get the next 50 results (page 2):

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "q": "advogado trabalhista",
    "location": "São Paulo",
    "hl": "pt-br",
    "gl": "br",
    "num": 50,
    "start": 50
  }'
```

### Full Example with All Options

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "q": "clínica odontológica",
    "location": "Belo Horizonte",
    "hl": "pt-br",
    "gl": "br",
    "exclude_domains": [
      "instagram.com",
      "linkedin.com",
      "facebook.com"
    ],
    "num": 30,
    "start": 0
  }'
```

### Example Response

```json
{
  "organic_results": [
    {
      "position": 1,
      "title": "Clínica Odontológica em BH - Sorriso Perfeito",
      "link": "https://www.sorrisoperfeito.com.br/",
      "displayed_link": "www.sorrisoperfeito.com.br",
      "snippet": "A melhor clínica odontológica de Belo Horizonte. Implantes, ortodontia, clareamento e mais. Agende sua consulta!",
      "rating": 4.8,
      "reviews": 245
    },
    {
      "position": 2,
      "title": "Odonto BH - Clínica Odontológica",
      "link": "https://www.odontobh.com.br/",
      "displayed_link": "www.odontobh.com.br",
      "snippet": "Tratamentos odontológicos com qualidade e preço justo. Venha conhecer nossa clínica."
    }
  ],
  "serpapi_pagination": {
    "current": 1,
    "next": "https://serpapi.com/search.json?engine=google&start=30..."
  }
}
```

### Error Response Example

When required fields are missing:

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "location": "Recife"
  }'
```

Response (400 Bad Request):
```json
{
  "error": "Key: 'SearchRequest.Q' Error:Field validation for 'Q' failed on the 'required' tag"
}
```

---

## Swagger Documentation

This project includes auto-generated **Swagger 2.0** (OpenAPI 2.0) documentation using [swaggo/swag](https://github.com/swaggo/swag).

### Accessing Swagger UI

When the server is running, visit:

```
http://localhost:8080/swagger/index.html
```

### Regenerating Swagger Docs

After modifying API annotations, regenerate the docs:

```bash
# Install swag CLI (if not already installed)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger 2.0 docs
swag init -g cmd/api/main.go -o docs
```

### Swagger Annotations

Annotations are added as comments in Go files:

**Main file (`cmd/api/main.go`):**
```go
// @title Lead Gen Worker API
// @version 1.0
// @description API description here

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
```

**Controller methods:**
```go
// @Summary      Search Google for leads
// @Description  Perform a Google search using SerpAPI
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        request body dto.SearchRequest true "Search parameters"
// @Success      200 {object} handlers.SearchResponse
// @Failure      400 {object} dto.ErrorResponse
// @Router       /search [post]
func (ctrl *SearchController) Search(c *gin.Context) {
    // ...
}
```

### Generated Files

After running `swag init`, the following files are generated in the `docs/` folder:

| File | Description |
|------|-------------|
| `docs.go` | Go file with embedded Swagger spec |
| `swagger.json` | Swagger 2.0 spec in JSON format |
| `swagger.yaml` | Swagger 2.0 spec in YAML format |

---

## How Domain Exclusion Works

The domain exclusion feature uses Google's `-site:` search operator. When you specify domains to exclude, the API automatically appends them to your search query.

For example, if you search for:
```json
{
  "q": "escritório de contabilidade",
  "exclude_domains": ["instagram.com", "linkedin.com"]
}
```

The actual query sent to Google becomes:
```
escritório de contabilidade -site:instagram.com -site:linkedin.com
```

---

## How Location Resolution Works

The API automatically resolves location names to their canonical form using SerpAPI's locations endpoint. This ensures accurate geo-targeted results.

For example:
- Input: `"Recife"`
- Resolved: `"Recife,State of Pernambuco,Brazil"`

---

## How Multi-Page Fetching Works

SerpAPI's Google Light engine returns **10 results per page** by default. When you request more results (e.g., `num: 50`), the API automatically fetches multiple pages and combines them into a single response.

### How it works

1. You request `num: 50` results
2. The API calculates that 5 pages are needed (50 ÷ 10 = 5)
3. It fetches pages sequentially: `start=0`, `start=10`, `start=20`, `start=30`, `start=40`
4. Results are combined and positions are renumbered sequentially (1-50)
5. Response includes metadata: `total_results` and `pages_fetched`

### Example Request

```json
{
  "q": "escritório de contabilidade",
  "location": "Recife",
  "hl": "pt-br",
  "gl": "br",
  "num": 50
}
```

### Example Response

```json
{
  "total_results": 50,
  "pages_fetched": 5,
  "organic_results": [
    {"position": 1, "title": "Result 1", ...},
    {"position": 2, "title": "Result 2", ...},
    ...
    {"position": 50, "title": "Result 50", ...}
  ],
  "serpapi_pagination": {
    "current": 5,
    "next": "https://serpapi.com/search?start=50..."
  }
}
```

### Limits

| Parameter | Default | Maximum | Description |
|-----------|---------|---------|-------------|
| `num` | 10 | 100 | Total results to return |
| Pages fetched | 1 | 10 | Maximum pages fetched per request |
| Results per page | 10 | 10 | Fixed by SerpAPI |

### Important Notes

- **API Credits**: Each page fetched consumes one SerpAPI credit. Requesting 50 results uses 5 credits.
- **Performance**: Fetching multiple pages takes longer. 50 results ≈ 5× the time of 10 results.
- **Early termination**: If SerpAPI has no more results, fetching stops early.
- **Error handling**: If a page fails after some results were fetched, partial results are returned.

---

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Run tests for a specific package
go test ./internal/api/... -v

# Run a specific test
go test ./internal/api/controllers -run TestSearch_Success -v

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Test Structure

```
lead-gen-worker/
├── internal/
│   ├── api/
│   │   ├── router.go
│   │   ├── router_test.go              # Router tests
│   │   └── controllers/
│   │       ├── search_controller.go
│   │       └── search_controller_test.go # Controller tests
│   ├── dto/
│   │   ├── search.go
│   │   └── search_test.go              # DTO tests
│   └── handlers/
│       ├── googleSearchHandler.go
│       └── googleSearchHandler_test.go  # Handler tests
```

### Test Categories

| Package | Tests | Description |
|---------|-------|-------------|
| `internal/api` | Router tests | Health check, route registration, 404 handling |
| `internal/api/controllers` | Controller tests | Request validation, response format, error handling |
| `internal/dto` | DTO tests | JSON marshaling/unmarshaling, field validation |
| `internal/handlers` | Handler tests | Helper functions, struct validation, business logic |

### Example Test Output

```bash
$ go test ./... -v
=== RUN   TestHealthCheck
--- PASS: TestHealthCheck (0.00s)
=== RUN   TestSearch_Success
--- PASS: TestSearch_Success (0.00s)
=== RUN   TestSearch_MissingRequiredField_Q
--- PASS: TestSearch_MissingRequiredField_Q (0.00s)
...
PASS
ok      webstar/noturno-leadgen-worker/internal/api           0.020s
ok      webstar/noturno-leadgen-worker/internal/api/controllers   0.006s
ok      webstar/noturno-leadgen-worker/internal/dto           0.005s
ok      webstar/noturno-leadgen-worker/internal/handlers      0.004s
```

### Writing New Tests

Tests use [testify](https://github.com/stretchr/testify) for assertions:

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    // require stops test on failure
    require.NoError(t, err)
    
    // assert continues test on failure
    assert.Equal(t, expected, actual)
    assert.Contains(t, str, substring)
    assert.Len(t, slice, 3)
}
```

For HTTP handler tests, use `httptest`:

```go
func TestEndpoint(t *testing.T) {
    router := setupTestRouter()
    
    req, _ := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Building for Different Platforms

**Linux:**
```bash
GOOS=linux GOARCH=amd64 go build -o bin/api-linux ./cmd/api
```

**macOS:**
```bash
GOOS=darwin GOARCH=amd64 go build -o bin/api-darwin ./cmd/api
```

**Windows:**
```bash
GOOS=windows GOARCH=amd64 go build -o bin/api.exe ./cmd/api
```

### Code Structure Guidelines

- **`cmd/`** - Application entry points
- **`internal/`** - Private application code (not importable by other projects)
  - **`api/`** - HTTP layer (router, controllers)
  - **`config/`** - Configuration management
  - **`dto/`** - Data Transfer Objects (request/response structures)
  - **`handlers/`** - Business logic handlers
- **`docs/`** - Auto-generated Swagger documentation

---

## Docker (Optional)

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /api .
EXPOSE 8080
CMD ["./api"]
```

### Build and Run

```bash
docker build -t lead-gen-worker .
docker run -p 8080:8080 -e SERPAPI_KEY="your-api-key" lead-gen-worker
```

---

## Rate Limiting & Best Practices

1. **SerpAPI Limits**: Be aware of your SerpAPI plan's rate limits and monthly search quota.

2. **Caching**: Consider implementing caching for repeated searches to reduce API calls.

3. **Error Handling**: Always handle errors gracefully in your client applications.

4. **Pagination**: Use pagination (`num` and `start`) efficiently. Maximum 100 results per request.

---

## Troubleshooting

### "SERPAPI_KEY environment variable is required"

Make sure you've set the `SERPAPI_KEY` environment variable before starting the application.

### "no location found for: X"

The location couldn't be resolved. Try using a more specific location name or check the spelling.

### "failed to fetch location"

Network error when connecting to SerpAPI's location endpoint. Check your internet connection.

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin) - HTTP web framework
- [SerpAPI](https://serpapi.com/) - Google Search API provider