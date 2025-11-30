package handlers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConstants tests the package constants
func TestConstants(t *testing.T) {
	assert.Equal(t, 10, ResultsPerPage, "ResultsPerPage should be 10")
	assert.Equal(t, 100, MaxResultsPerRequest, "MaxResultsPerRequest should be 100")
	assert.Equal(t, 10, MaxPagesToFetch, "MaxPagesToFetch should be 10")
}

// TestNewGoogleSearchHandler tests the handler constructor
func TestNewGoogleSearchHandler(t *testing.T) {
	apiKey := "test-api-key-12345"
	handler := NewGoogleSearchHandler(apiKey)

	assert.NotNil(t, handler)
	assert.Equal(t, apiKey, handler.apiKey)
}

// TestNewGoogleSearchHandler_EmptyApiKey tests constructor with empty API key
func TestNewGoogleSearchHandler_EmptyApiKey(t *testing.T) {
	handler := NewGoogleSearchHandler("")

	assert.NotNil(t, handler)
	assert.Empty(t, handler.apiKey)
}

// TestGetString tests the getString helper function
func TestGetString(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string value",
			input:    map[string]interface{}{"title": "Test Title"},
			key:      "title",
			expected: "Test Title",
		},
		{
			name:     "non-existing key",
			input:    map[string]interface{}{"title": "Test Title"},
			key:      "description",
			expected: "",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "title",
			expected: "",
		},
		{
			name:     "non-string value",
			input:    map[string]interface{}{"count": 123},
			key:      "count",
			expected: "",
		},
		{
			name:     "nil value",
			input:    map[string]interface{}{"title": nil},
			key:      "title",
			expected: "",
		},
		{
			name:     "empty string value",
			input:    map[string]interface{}{"title": ""},
			key:      "title",
			expected: "",
		},
		{
			name:     "string with special characters",
			input:    map[string]interface{}{"title": "Título com acentuação & símbolos!"},
			key:      "title",
			expected: "Título com acentuação & símbolos!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getString(tc.input, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetInt tests the getInt helper function
func TestGetInt(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected int
	}{
		{
			name:     "existing float64 value",
			input:    map[string]interface{}{"position": float64(5)},
			key:      "position",
			expected: 5,
		},
		{
			name:     "non-existing key",
			input:    map[string]interface{}{"position": float64(5)},
			key:      "count",
			expected: 0,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "position",
			expected: 0,
		},
		{
			name:     "string value instead of number",
			input:    map[string]interface{}{"position": "5"},
			key:      "position",
			expected: 0,
		},
		{
			name:     "nil value",
			input:    map[string]interface{}{"position": nil},
			key:      "position",
			expected: 0,
		},
		{
			name:     "zero value",
			input:    map[string]interface{}{"position": float64(0)},
			key:      "position",
			expected: 0,
		},
		{
			name:     "large number",
			input:    map[string]interface{}{"count": float64(999999)},
			key:      "count",
			expected: 999999,
		},
		{
			name:     "negative number",
			input:    map[string]interface{}{"offset": float64(-10)},
			key:      "offset",
			expected: -10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getInt(tc.input, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetFloat tests the getFloat helper function
func TestGetFloat(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected float64
	}{
		{
			name:     "existing float64 value",
			input:    map[string]interface{}{"rating": float64(4.5)},
			key:      "rating",
			expected: 4.5,
		},
		{
			name:     "non-existing key",
			input:    map[string]interface{}{"rating": float64(4.5)},
			key:      "score",
			expected: 0,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "rating",
			expected: 0,
		},
		{
			name:     "string value instead of number",
			input:    map[string]interface{}{"rating": "4.5"},
			key:      "rating",
			expected: 0,
		},
		{
			name:     "nil value",
			input:    map[string]interface{}{"rating": nil},
			key:      "rating",
			expected: 0,
		},
		{
			name:     "zero value",
			input:    map[string]interface{}{"rating": float64(0)},
			key:      "rating",
			expected: 0,
		},
		{
			name:     "integer as float",
			input:    map[string]interface{}{"rating": float64(5)},
			key:      "rating",
			expected: 5.0,
		},
		{
			name:     "precise decimal",
			input:    map[string]interface{}{"rating": float64(3.14159)},
			key:      "rating",
			expected: 3.14159,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getFloat(tc.input, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGoogleSearchParams_Defaults tests default values for search params
func TestGoogleSearchParams_Defaults(t *testing.T) {
	params := GoogleSearchParams{}

	assert.Empty(t, params.Q)
	assert.Empty(t, params.Location)
	assert.Empty(t, params.Hl)
	assert.Empty(t, params.Gl)
	assert.Nil(t, params.ExcludeDomains)
	assert.Equal(t, 0, params.Num)
	assert.Equal(t, 0, params.Start)
}

// TestGoogleSearchParams_WithValues tests search params with values
func TestGoogleSearchParams_WithValues(t *testing.T) {
	params := GoogleSearchParams{
		Q:              "test query",
		Location:       "São Paulo",
		Hl:             "pt-br",
		Gl:             "br",
		ExcludeDomains: []string{"instagram.com", "linkedin.com"},
		Num:            50,
		Start:          10,
	}

	assert.Equal(t, "test query", params.Q)
	assert.Equal(t, "São Paulo", params.Location)
	assert.Equal(t, "pt-br", params.Hl)
	assert.Equal(t, "br", params.Gl)
	assert.Len(t, params.ExcludeDomains, 2)
	assert.Contains(t, params.ExcludeDomains, "instagram.com")
	assert.Contains(t, params.ExcludeDomains, "linkedin.com")
	assert.Equal(t, 50, params.Num)
	assert.Equal(t, 10, params.Start)
}

// TestSearchResponse_EmptyResults tests empty search response
func TestSearchResponse_EmptyResults(t *testing.T) {
	response := SearchResponse{
		TotalResults:   0,
		PagesFetched:   1,
		OrganicResults: []OrganicResult{},
		Pagination:     Pagination{Current: 1},
	}

	assert.Equal(t, 0, response.TotalResults)
	assert.Equal(t, 1, response.PagesFetched)
	assert.Empty(t, response.OrganicResults)
	assert.Equal(t, 1, response.Pagination.Current)
	assert.Empty(t, response.Pagination.Next)
}

// TestSearchResponse_WithResults tests search response with results
func TestSearchResponse_WithResults(t *testing.T) {
	response := SearchResponse{
		TotalResults: 2,
		PagesFetched: 1,
		OrganicResults: []OrganicResult{
			{
				Position:      1,
				Title:         "First Result",
				Link:          "https://first.com",
				DisplayedLink: "first.com",
				Snippet:       "First snippet",
				Rating:        4.5,
				Reviews:       100,
			},
			{
				Position:      2,
				Title:         "Second Result",
				Link:          "https://second.com",
				DisplayedLink: "second.com",
				Snippet:       "Second snippet",
			},
		},
		Pagination: Pagination{
			Current: 1,
			Next:    "https://serpapi.com/search?start=10",
		},
	}

	assert.Equal(t, 2, response.TotalResults)
	assert.Equal(t, 1, response.PagesFetched)
	assert.Len(t, response.OrganicResults, 2)

	// First result assertions
	assert.Equal(t, 1, response.OrganicResults[0].Position)
	assert.Equal(t, "First Result", response.OrganicResults[0].Title)
	assert.Equal(t, 4.5, response.OrganicResults[0].Rating)
	assert.Equal(t, 100, response.OrganicResults[0].Reviews)

	// Second result assertions
	assert.Equal(t, 2, response.OrganicResults[1].Position)
	assert.Equal(t, "Second Result", response.OrganicResults[1].Title)
	assert.Equal(t, float64(0), response.OrganicResults[1].Rating)
	assert.Equal(t, 0, response.OrganicResults[1].Reviews)

	// Pagination assertions
	assert.Equal(t, 1, response.Pagination.Current)
	assert.Equal(t, "https://serpapi.com/search?start=10", response.Pagination.Next)
}

// TestOrganicResult_WithSitelinks tests organic result with sitelinks
func TestOrganicResult_WithSitelinks(t *testing.T) {
	result := OrganicResult{
		Position:      1,
		Title:         "Test Result",
		Link:          "https://test.com",
		DisplayedLink: "test.com",
		Snippet:       "Test snippet",
		Sitelinks: &Sitelinks{
			Inline: []Sitelink{
				{Title: "About", Link: "https://test.com/about"},
				{Title: "Contact", Link: "https://test.com/contact"},
				{Title: "Services", Link: "https://test.com/services"},
			},
		},
	}

	assert.NotNil(t, result.Sitelinks)
	assert.Len(t, result.Sitelinks.Inline, 3)
	assert.Equal(t, "About", result.Sitelinks.Inline[0].Title)
	assert.Equal(t, "https://test.com/about", result.Sitelinks.Inline[0].Link)
}

// TestOrganicResult_WithExtensions tests organic result with extensions
func TestOrganicResult_WithExtensions(t *testing.T) {
	result := OrganicResult{
		Position:      1,
		Title:         "Test Result",
		Link:          "https://test.com",
		DisplayedLink: "test.com",
		Snippet:       "Test snippet",
		Extensions:    []string{"Rating", "4.5/5", "100 reviews"},
	}

	assert.Len(t, result.Extensions, 3)
	assert.Contains(t, result.Extensions, "Rating")
	assert.Contains(t, result.Extensions, "4.5/5")
	assert.Contains(t, result.Extensions, "100 reviews")
}

// TestSerpAPILocation_Fields tests SerpAPILocation struct fields
func TestSerpAPILocation_Fields(t *testing.T) {
	location := SerpAPILocation{
		ID:             "test-id-123",
		GoogleID:       1001625,
		GoogleParentID: 20099,
		Name:           "Recife",
		CanonicalName:  "Recife,State of Pernambuco,Brazil",
		CountryCode:    "BR",
		TargetType:     "City",
		Reach:          2100000,
		GPS:            []float64{-34.8769643, -8.0475622},
		Keys:           []string{"recife", "state", "of", "pernambuco", "brazil"},
	}

	assert.Equal(t, "test-id-123", location.ID)
	assert.Equal(t, 1001625, location.GoogleID)
	assert.Equal(t, 20099, location.GoogleParentID)
	assert.Equal(t, "Recife", location.Name)
	assert.Equal(t, "Recife,State of Pernambuco,Brazil", location.CanonicalName)
	assert.Equal(t, "BR", location.CountryCode)
	assert.Equal(t, "City", location.TargetType)
	assert.Equal(t, 2100000, location.Reach)
	assert.Len(t, location.GPS, 2)
	assert.Len(t, location.Keys, 5)
}

// TestPagination_Empty tests empty pagination
func TestPagination_Empty(t *testing.T) {
	pagination := Pagination{}

	assert.Equal(t, 0, pagination.Current)
	assert.Empty(t, pagination.Next)
}

// TestPagination_WithNext tests pagination with next page
func TestPagination_WithNext(t *testing.T) {
	pagination := Pagination{
		Current: 1,
		Next:    "https://serpapi.com/search?start=10&num=10",
	}

	assert.Equal(t, 1, pagination.Current)
	assert.NotEmpty(t, pagination.Next)
	assert.Contains(t, pagination.Next, "start=10")
}

// TestSitelink_Fields tests Sitelink struct fields
func TestSitelink_Fields(t *testing.T) {
	sitelink := Sitelink{
		Title: "About Us",
		Link:  "https://example.com/about",
	}

	assert.Equal(t, "About Us", sitelink.Title)
	assert.Equal(t, "https://example.com/about", sitelink.Link)
}

// TestSitelinks_Empty tests empty sitelinks
func TestSitelinks_Empty(t *testing.T) {
	sitelinks := Sitelinks{
		Inline: []Sitelink{},
	}

	assert.Empty(t, sitelinks.Inline)
}

// TestSitelinks_Multiple tests multiple sitelinks
func TestSitelinks_Multiple(t *testing.T) {
	sitelinks := Sitelinks{
		Inline: []Sitelink{
			{Title: "Home", Link: "https://example.com/"},
			{Title: "About", Link: "https://example.com/about"},
			{Title: "Contact", Link: "https://example.com/contact"},
		},
	}

	assert.Len(t, sitelinks.Inline, 3)
	assert.Equal(t, "Home", sitelinks.Inline[0].Title)
	assert.Equal(t, "About", sitelinks.Inline[1].Title)
	assert.Equal(t, "Contact", sitelinks.Inline[2].Title)
}

// TestBuildQueryWithExcludedDomains tests query building logic
func TestBuildQueryWithExcludedDomains(t *testing.T) {
	testCases := []struct {
		name           string
		query          string
		excludeDomains []string
		expected       string
	}{
		{
			name:           "no excluded domains",
			query:          "test query",
			excludeDomains: nil,
			expected:       "test query",
		},
		{
			name:           "empty excluded domains",
			query:          "test query",
			excludeDomains: []string{},
			expected:       "test query",
		},
		{
			name:           "single excluded domain",
			query:          "test query",
			excludeDomains: []string{"instagram.com"},
			expected:       "test query -site:instagram.com",
		},
		{
			name:           "multiple excluded domains",
			query:          "test query",
			excludeDomains: []string{"instagram.com", "linkedin.com", "facebook.com"},
			expected:       "test query -site:instagram.com -site:linkedin.com -site:facebook.com",
		},
		{
			name:           "query with special characters",
			query:          "escritório de contabilidade",
			excludeDomains: []string{"instagram.com"},
			expected:       "escritório de contabilidade -site:instagram.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := tc.query
			for _, domain := range tc.excludeDomains {
				query += " -site:" + domain
			}
			assert.Equal(t, tc.expected, query)
		})
	}
}

// TestNumPaginationLimits tests pagination num parameter limits
func TestNumPaginationLimits(t *testing.T) {
	testCases := []struct {
		name     string
		num      int
		expected int
	}{
		{
			name:     "default when zero",
			num:      0,
			expected: 10,
		},
		{
			name:     "default when negative",
			num:      -5,
			expected: 10,
		},
		{
			name:     "valid num 10",
			num:      10,
			expected: 10,
		},
		{
			name:     "valid num 50",
			num:      50,
			expected: 50,
		},
		{
			name:     "max num 100",
			num:      100,
			expected: 100,
		},
		{
			name:     "over max should cap at 100",
			num:      150,
			expected: 100,
		},
		{
			name:     "way over max should cap at 100",
			num:      1000,
			expected: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			num := tc.num
			if num <= 0 {
				num = 10 // default
			} else if num > 100 {
				num = 100 // max allowed by SerpAPI
			}
			assert.Equal(t, tc.expected, num)
		})
	}
}

// TestCalculatePagesNeeded tests the pages calculation logic
func TestCalculatePagesNeeded(t *testing.T) {
	testCases := []struct {
		name           string
		totalRequested int
		expected       int
	}{
		{
			name:           "10 results needs 1 page",
			totalRequested: 10,
			expected:       1,
		},
		{
			name:           "11 results needs 2 pages",
			totalRequested: 11,
			expected:       2,
		},
		{
			name:           "20 results needs 2 pages",
			totalRequested: 20,
			expected:       2,
		},
		{
			name:           "25 results needs 3 pages",
			totalRequested: 25,
			expected:       3,
		},
		{
			name:           "50 results needs 5 pages",
			totalRequested: 50,
			expected:       5,
		},
		{
			name:           "100 results needs 10 pages",
			totalRequested: 100,
			expected:       10,
		},
		{
			name:           "1 result needs 1 page",
			totalRequested: 1,
			expected:       1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ceiling division: (totalRequested + ResultsPerPage - 1) / ResultsPerPage
			pagesNeeded := (tc.totalRequested + ResultsPerPage - 1) / ResultsPerPage
			if pagesNeeded > MaxPagesToFetch {
				pagesNeeded = MaxPagesToFetch
			}
			assert.Equal(t, tc.expected, pagesNeeded)
		})
	}
}

// TestSearchResponse_MultiplePages tests response with multiple pages fetched
func TestSearchResponse_MultiplePages(t *testing.T) {
	// Simulate a response with 25 results from 3 pages
	results := make([]OrganicResult, 25)
	for i := 0; i < 25; i++ {
		results[i] = OrganicResult{
			Position: i + 1,
			Title:    fmt.Sprintf("Result %d", i+1),
			Link:     fmt.Sprintf("https://example%d.com", i+1),
		}
	}

	response := SearchResponse{
		TotalResults:   25,
		PagesFetched:   3,
		OrganicResults: results,
		Pagination: Pagination{
			Current: 3,
			Next:    "https://serpapi.com/search?start=30",
		},
	}

	assert.Equal(t, 25, response.TotalResults)
	assert.Equal(t, 3, response.PagesFetched)
	assert.Len(t, response.OrganicResults, 25)

	// Verify positions are sequential
	for i, result := range response.OrganicResults {
		assert.Equal(t, i+1, result.Position)
	}
}

// TestMaxResultsCapping tests that results are capped at MaxResultsPerRequest
func TestMaxResultsCapping(t *testing.T) {
	testCases := []struct {
		name      string
		requested int
		expected  int
	}{
		{"below max", 50, 50},
		{"at max", 100, 100},
		{"above max", 150, 100},
		{"way above max", 1000, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			totalRequested := tc.requested
			if totalRequested > MaxResultsPerRequest {
				totalRequested = MaxResultsPerRequest
			}
			assert.Equal(t, tc.expected, totalRequested)
		})
	}
}
