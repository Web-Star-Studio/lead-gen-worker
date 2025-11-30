package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter creates a Gin router for testing
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// MockSearchHandler is a mock implementation for testing
type MockSearchHandler struct {
	MockResponse *handlers.SearchResponse
	MockError    error
}

func (m *MockSearchHandler) Search(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
	if m.MockError != nil {
		return nil, m.MockError
	}
	return m.MockResponse, nil
}

// MockableSearchController allows injecting a mock search function
type MockableSearchController struct {
	searchFunc func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error)
}

func NewMockableSearchController(searchFunc func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error)) *MockableSearchController {
	return &MockableSearchController{searchFunc: searchFunc}
}

func (ctrl *MockableSearchController) Search(c *gin.Context) {
	var req dto.SearchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	params := handlers.GoogleSearchParams{
		Q:              req.Q,
		Location:       req.Location,
		Hl:             req.Hl,
		Gl:             req.Gl,
		ExcludeDomains: req.ExcludeDomains,
		Num:            req.Num,
		Start:          req.Start,
	}

	result, err := ctrl.searchFunc(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// TestSearch_Success tests a successful search request
func TestSearch_Success(t *testing.T) {
	router := setupTestRouter()

	mockResponse := &handlers.SearchResponse{
		OrganicResults: []handlers.OrganicResult{
			{
				Position:      1,
				Title:         "Test Result",
				Link:          "https://example.com",
				DisplayedLink: "example.com",
				Snippet:       "This is a test snippet",
				Rating:        4.5,
				Reviews:       100,
			},
		},
		Pagination: handlers.Pagination{
			Current: 1,
			Next:    "https://serpapi.com/search?start=10",
		},
	}

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		// Verify params are passed correctly
		assert.Equal(t, "test query", params.Q)
		assert.Equal(t, "São Paulo", params.Location)
		assert.Equal(t, "pt-br", params.Hl)
		assert.Equal(t, "br", params.Gl)
		assert.Equal(t, []string{"instagram.com", "linkedin.com"}, params.ExcludeDomains)
		assert.Equal(t, 20, params.Num)
		assert.Equal(t, 0, params.Start)
		return mockResponse, nil
	})

	router.POST("/api/v1/search", controller.Search)

	requestBody := dto.SearchRequest{
		Q:              "test query",
		Location:       "São Paulo",
		Hl:             "pt-br",
		Gl:             "br",
		ExcludeDomains: []string{"instagram.com", "linkedin.com"},
		Num:            20,
		Start:          0,
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.SearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.OrganicResults, 1)
	assert.Equal(t, "Test Result", response.OrganicResults[0].Title)
	assert.Equal(t, "https://example.com", response.OrganicResults[0].Link)
	assert.Equal(t, 1, response.Pagination.Current)
}

// TestSearch_MissingRequiredField_Q tests validation error when 'q' field is missing
func TestSearch_MissingRequiredField_Q(t *testing.T) {
	router := setupTestRouter()

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		t.Fatal("Search should not be called when validation fails")
		return nil, nil
	})

	router.POST("/api/v1/search", controller.Search)

	// Missing 'q' field
	requestBody := map[string]interface{}{
		"location": "Recife",
		"hl":       "pt-br",
		"gl":       "br",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse dto.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.Contains(t, errorResponse.Error, "SearchRequest.Q")
	assert.Contains(t, errorResponse.Error, "required")
}

// TestSearch_MissingRequiredField_Location tests validation error when 'location' field is missing
func TestSearch_MissingRequiredField_Location(t *testing.T) {
	router := setupTestRouter()

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		t.Fatal("Search should not be called when validation fails")
		return nil, nil
	})

	router.POST("/api/v1/search", controller.Search)

	// Missing 'location' field
	requestBody := map[string]interface{}{
		"q":  "test query",
		"hl": "pt-br",
		"gl": "br",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse dto.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.Contains(t, errorResponse.Error, "SearchRequest.Location")
	assert.Contains(t, errorResponse.Error, "required")
}

// TestSearch_InvalidJSON tests validation error for malformed JSON
func TestSearch_InvalidJSON(t *testing.T) {
	router := setupTestRouter()

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		t.Fatal("Search should not be called when validation fails")
		return nil, nil
	})

	router.POST("/api/v1/search", controller.Search)

	// Invalid JSON
	invalidJSON := []byte(`{"q": "test", "location": }`)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(invalidJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse dto.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, errorResponse.Error)
}

// TestSearch_EmptyBody tests validation error for empty request body
func TestSearch_EmptyBody(t *testing.T) {
	router := setupTestRouter()

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		t.Fatal("Search should not be called when validation fails")
		return nil, nil
	})

	router.POST("/api/v1/search", controller.Search)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer([]byte{}))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSearch_InternalServerError tests handling of internal errors
func TestSearch_InternalServerError(t *testing.T) {
	router := setupTestRouter()

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		return nil, assert.AnError // Return a generic error
	})

	router.POST("/api/v1/search", controller.Search)

	requestBody := dto.SearchRequest{
		Q:        "test query",
		Location: "Recife",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse dto.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, errorResponse.Error)
}

// TestSearch_WithOptionalFields tests that optional fields are properly handled
func TestSearch_WithOptionalFields(t *testing.T) {
	router := setupTestRouter()

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		// Verify optional fields have default/zero values when not provided
		assert.Equal(t, "test query", params.Q)
		assert.Equal(t, "Recife", params.Location)
		assert.Empty(t, params.Hl)
		assert.Empty(t, params.Gl)
		assert.Nil(t, params.ExcludeDomains)
		assert.Equal(t, 0, params.Num)
		assert.Equal(t, 0, params.Start)
		return &handlers.SearchResponse{}, nil
	})

	router.POST("/api/v1/search", controller.Search)

	// Only required fields
	requestBody := dto.SearchRequest{
		Q:        "test query",
		Location: "Recife",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSearch_WithPagination tests pagination parameters
func TestSearch_WithPagination(t *testing.T) {
	testCases := []struct {
		name     string
		num      int
		start    int
		expected int
	}{
		{"first page with 10 results", 10, 0, 10},
		{"second page with 10 results", 10, 10, 10},
		{"50 results first page", 50, 0, 50},
		{"100 results (max)", 100, 0, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupTestRouter()

			controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
				assert.Equal(t, tc.num, params.Num)
				assert.Equal(t, tc.start, params.Start)
				return &handlers.SearchResponse{}, nil
			})

			router.POST("/api/v1/search", controller.Search)

			requestBody := dto.SearchRequest{
				Q:        "test query",
				Location: "Recife",
				Num:      tc.num,
				Start:    tc.start,
			}

			body, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestSearch_WithExcludeDomains tests domain exclusion feature
func TestSearch_WithExcludeDomains(t *testing.T) {
	router := setupTestRouter()

	excludeDomains := []string{"instagram.com", "linkedin.com", "facebook.com", "twitter.com"}

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		assert.Equal(t, excludeDomains, params.ExcludeDomains)
		assert.Len(t, params.ExcludeDomains, 4)
		return &handlers.SearchResponse{}, nil
	})

	router.POST("/api/v1/search", controller.Search)

	requestBody := dto.SearchRequest{
		Q:              "test query",
		Location:       "Recife",
		ExcludeDomains: excludeDomains,
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSearch_ResponseStructure tests that the response has the expected structure
func TestSearch_ResponseStructure(t *testing.T) {
	router := setupTestRouter()

	mockResponse := &handlers.SearchResponse{
		OrganicResults: []handlers.OrganicResult{
			{
				Position:      1,
				Title:         "First Result",
				Link:          "https://first.com",
				DisplayedLink: "first.com",
				Snippet:       "First snippet",
				Rating:        4.5,
				Reviews:       100,
				Extensions:    []string{"Rating", "4.5/5"},
				Sitelinks: &handlers.Sitelinks{
					Inline: []handlers.Sitelink{
						{Title: "About", Link: "https://first.com/about"},
						{Title: "Contact", Link: "https://first.com/contact"},
					},
				},
			},
			{
				Position:      2,
				Title:         "Second Result",
				Link:          "https://second.com",
				DisplayedLink: "second.com",
				Snippet:       "Second snippet",
			},
		},
		Pagination: handlers.Pagination{
			Current: 1,
			Next:    "https://serpapi.com/search?start=10",
		},
	}

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		return mockResponse, nil
	})

	router.POST("/api/v1/search", controller.Search)

	requestBody := dto.SearchRequest{
		Q:        "test query",
		Location: "Recife",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.SearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify organic_results
	assert.Len(t, response.OrganicResults, 2)

	// First result with sitelinks
	first := response.OrganicResults[0]
	assert.Equal(t, 1, first.Position)
	assert.Equal(t, "First Result", first.Title)
	assert.Equal(t, "https://first.com", first.Link)
	assert.Equal(t, 4.5, first.Rating)
	assert.Equal(t, 100, first.Reviews)
	assert.NotNil(t, first.Sitelinks)
	assert.Len(t, first.Sitelinks.Inline, 2)
	assert.Equal(t, "About", first.Sitelinks.Inline[0].Title)

	// Second result without sitelinks
	second := response.OrganicResults[1]
	assert.Equal(t, 2, second.Position)
	assert.Nil(t, second.Sitelinks)

	// Verify pagination
	assert.Equal(t, 1, response.Pagination.Current)
	assert.Equal(t, "https://serpapi.com/search?start=10", response.Pagination.Next)
}

// TestSearch_ContentTypeHeader tests that the response has correct content type
func TestSearch_ContentTypeHeader(t *testing.T) {
	router := setupTestRouter()

	controller := NewMockableSearchController(func(params handlers.GoogleSearchParams) (*handlers.SearchResponse, error) {
		return &handlers.SearchResponse{}, nil
	})

	router.POST("/api/v1/search", controller.Search)

	requestBody := dto.SearchRequest{
		Q:        "test query",
		Location: "Recife",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}
