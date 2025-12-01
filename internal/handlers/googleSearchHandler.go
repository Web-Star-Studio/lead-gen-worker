package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"webstar/noturno-leadgen-worker/internal/dto"

	g "github.com/serpapi/google-search-results-golang"
)

const (
	// ResultsPerPage is the number of results SerpAPI returns per page
	ResultsPerPage = 10
	// MaxResultsPerRequest is the maximum results we allow per request
	MaxResultsPerRequest = 100
	// MaxPagesToFetch is the maximum number of pages we'll fetch to prevent excessive API calls
	MaxPagesToFetch = 10
)

type GoogleSearchHandler struct {
	apiKey               string
	params               GoogleSearchParams
	firecrawlHandler     *FirecrawlHandler
	dataExtractorHandler *DataExtractorHandler
	preCallReportHandler *PreCallReportHandler
	coldEmailHandler     *ColdEmailHandler
}

type GoogleSearchParams struct {
	Q              string
	Location       string
	Hl             string   // language used in the query
	Gl             string   // country to use for the search
	ExcludeDomains []string // domains to exclude from search results (e.g., "instagram.com", "linkedin.com")
	Num            int      // total number of results to return (will fetch multiple pages if needed)
	Start          int      // result offset for pagination (0 = first page)
}

// Sitelink represents an inline sitelink in organic results
// @Description Inline sitelink with title and URL
type Sitelink struct {
	// Title of the sitelink
	Title string `json:"title" example:"Contact Us"`
	// URL of the sitelink
	Link string `json:"link" example:"https://example.com/contact"`
}

// Sitelinks contains the sitelinks for an organic result
// @Description Container for inline sitelinks
type Sitelinks struct {
	// List of inline sitelinks
	Inline []Sitelink `json:"inline,omitempty"`
}

// OrganicResult represents a single organic search result
// @Description A single organic search result from Google
type OrganicResult struct {
	// Position of the result in the search results
	Position int `json:"position" example:"1"`
	// Title of the search result
	Title string `json:"title" example:"Escritório de Contabilidade em Recife - Contador em Recife"`
	// URL of the search result
	Link string `json:"link" example:"https://www.example.com.br/"`
	// Displayed URL shown in search results
	DisplayedLink string `json:"displayed_link" example:"www.example.com.br"`
	// Snippet/description of the search result
	Snippet string `json:"snippet" example:"Precisa de um contador em Recife? Oferecemos serviços de contabilidade, abertura de empresa e muito mais."`
	// Additional extensions like ratings text
	Extensions []string `json:"extensions,omitempty" example:"Classificação,10/10 (3)"`
	// Rating score if available
	Rating float64 `json:"rating,omitempty" example:"4.8"`
	// Number of reviews if available
	Reviews int `json:"reviews,omitempty" example:"245"`
	// Sitelinks associated with this result
	Sitelinks *Sitelinks `json:"sitelinks,omitempty"`
	// ScrapedContent is the markdown content scraped from the website homepage (populated by FirecrawlHandler)
	ScrapedContent string `json:"scraped_content,omitempty"`
	// ScrapeError contains error message if scraping failed
	ScrapeError string `json:"scrape_error,omitempty"`
	// ExtractedData contains structured company data extracted by DataExtractorHandler
	ExtractedData *ExtractedData `json:"extracted_data,omitempty"`
	// PreCallReport contains the AI-generated company summary for sales calls
	PreCallReport string `json:"pre_call_report,omitempty"`
	// ColdEmail contains the AI-generated cold email for first contact
	ColdEmail *ColdEmail `json:"cold_email,omitempty"`
}

// Pagination represents the pagination info from SerpAPI
// @Description Pagination information for search results
type Pagination struct {
	// Current page number
	Current int `json:"current" example:"1"`
	// URL for the next page of results
	Next string `json:"next,omitempty" example:"https://serpapi.com/search.json?engine=google&start=10"`
}

// SearchResponse contains only organic_results and pagination
// @Description Response containing organic search results and pagination info
type SearchResponse struct {
	// Total number of results returned
	TotalResults int `json:"total_results" example:"50"`
	// Number of pages fetched to get these results
	PagesFetched int `json:"pages_fetched" example:"5"`
	// List of organic search results
	OrganicResults []OrganicResult `json:"organic_results"`
	// Pagination information (for the last page fetched)
	Pagination Pagination `json:"serpapi_pagination"`
}

// SerpAPILocation represents the location response from SerpAPI
type SerpAPILocation struct {
	ID             string    `json:"id"`
	GoogleID       int       `json:"google_id"`
	GoogleParentID int       `json:"google_parent_id"`
	Name           string    `json:"name"`
	CanonicalName  string    `json:"canonical_name"`
	CountryCode    string    `json:"country_code"`
	TargetType     string    `json:"target_type"`
	Reach          int       `json:"reach"`
	GPS            []float64 `json:"gps"`
	Keys           []string  `json:"keys"`
}

func NewGoogleSearchHandler(apiKey string) *GoogleSearchHandler {
	return &GoogleSearchHandler{
		apiKey: apiKey,
	}
}

// SetFirecrawlHandler sets the FirecrawlHandler for automatic website scraping
// When set, the Search method will automatically scrape each organic result's website
func (h *GoogleSearchHandler) SetFirecrawlHandler(handler *FirecrawlHandler) {
	h.firecrawlHandler = handler
}

// SetDataExtractorHandler sets the DataExtractorHandler for extracting company data
// When set, the Search method will automatically extract structured data from scraped content
func (h *GoogleSearchHandler) SetDataExtractorHandler(handler *DataExtractorHandler) {
	h.dataExtractorHandler = handler
}

// SetPreCallReportHandler sets the PreCallReportHandler for generating pre-call reports
// When set, the Search method will automatically generate AI-powered pre-call reports for each result
func (h *GoogleSearchHandler) SetPreCallReportHandler(handler *PreCallReportHandler) {
	h.preCallReportHandler = handler
}

// SetColdEmailHandler sets the ColdEmailHandler for generating cold emails
// When set, the Search method will automatically generate AI-powered cold emails for each result
func (h *GoogleSearchHandler) SetColdEmailHandler(handler *ColdEmailHandler) {
	h.coldEmailHandler = handler
}

// SetBusinessProfile sets the business profile on AI handlers for personalized content
func (h *GoogleSearchHandler) SetBusinessProfile(profile *dto.BusinessProfile) {
	if h.preCallReportHandler != nil {
		h.preCallReportHandler.SetBusinessProfile(profile)
	}
	if h.coldEmailHandler != nil {
		h.coldEmailHandler.SetBusinessProfile(profile)
	}
}

// ClearBusinessProfile clears the business profile from AI handlers
func (h *GoogleSearchHandler) ClearBusinessProfile() {
	if h.preCallReportHandler != nil {
		h.preCallReportHandler.ClearBusinessProfile()
	}
	if h.coldEmailHandler != nil {
		h.coldEmailHandler.ClearBusinessProfile()
	}
}

// getCanonicalLocation fetches the canonical location name from SerpAPI
func (h *GoogleSearchHandler) getCanonicalLocation(location string) (string, error) {
	// URL encode the location parameter
	encodedLocation := url.QueryEscape(location)
	requestURL := fmt.Sprintf("https://serpapi.com/locations.json?q=%s&limit=1", encodedLocation)

	log.Printf("[GoogleSearchHandler] Fetching canonical location for: %s", location)

	resp, err := http.Get(requestURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch location: %w", err)
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("location API returned status %d: %s", resp.StatusCode, string(body))
	}

	var locations []SerpAPILocation
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return "", fmt.Errorf("failed to decode location response: %w", err)
	}

	if len(locations) == 0 {
		// Fallback: use the original location string if no canonical found
		log.Printf("[GoogleSearchHandler] No canonical location found, using original: %s", location)
		return location, nil
	}

	log.Printf("[GoogleSearchHandler] Resolved location: %s -> %s", location, locations[0].CanonicalName)
	return locations[0].CanonicalName, nil
}

// fetchPage fetches a single page of results from SerpAPI
func (h *GoogleSearchHandler) fetchPage(query, canonicalLocation, hl, gl string, start int) ([]OrganicResult, *Pagination, error) {
	parameters := map[string]string{
		"engine":   "google",
		"q":        query,
		"location": canonicalLocation,
		"hl":       hl,
		"gl":       gl,
		"num":      fmt.Sprintf("%d", ResultsPerPage),
		"start":    fmt.Sprintf("%d", start),
	}

	search := g.NewGoogleSearch(parameters, h.apiKey)
	resp, err := search.GetJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch page at start=%d: %w", start, err)
	}

	var results []OrganicResult
	var pagination *Pagination

	// Parse organic_results
	if organicResults, ok := resp["organic_results"].([]interface{}); ok {
		for _, item := range organicResults {
			if itemMap, ok := item.(map[string]interface{}); ok {
				organic := OrganicResult{
					Position:      getInt(itemMap, "position"),
					Title:         getString(itemMap, "title"),
					Link:          getString(itemMap, "link"),
					DisplayedLink: getString(itemMap, "displayed_link"),
					Snippet:       getString(itemMap, "snippet"),
					Rating:        getFloat(itemMap, "rating"),
					Reviews:       getInt(itemMap, "reviews"),
				}

				// Parse extensions
				if extensions, ok := itemMap["extensions"].([]interface{}); ok {
					for _, ext := range extensions {
						if extStr, ok := ext.(string); ok {
							organic.Extensions = append(organic.Extensions, extStr)
						}
					}
				}

				// Parse sitelinks
				if sitelinksMap, ok := itemMap["sitelinks"].(map[string]interface{}); ok {
					organic.Sitelinks = &Sitelinks{}
					if inline, ok := sitelinksMap["inline"].([]interface{}); ok {
						for _, sl := range inline {
							if slMap, ok := sl.(map[string]interface{}); ok {
								organic.Sitelinks.Inline = append(organic.Sitelinks.Inline, Sitelink{
									Title: getString(slMap, "title"),
									Link:  getString(slMap, "link"),
								})
							}
						}
					}
				}

				results = append(results, organic)
			}
		}
	}

	// Parse serpapi_pagination
	if paginationMap, ok := resp["serpapi_pagination"].(map[string]interface{}); ok {
		pagination = &Pagination{
			Current: getInt(paginationMap, "current"),
			Next:    getString(paginationMap, "next"),
		}
	}

	return results, pagination, nil
}

// Search performs a Google search and fetches multiple pages if needed to meet the requested number of results
func (h *GoogleSearchHandler) Search(params GoogleSearchParams) (*SearchResponse, error) {
	// Get the canonical location name
	canonicalLocation, err := h.getCanonicalLocation(params.Location)
	if err != nil {
		return nil, err
	}

	// Build query with excluded domains
	query := params.Q
	for _, domain := range params.ExcludeDomains {
		query += " -site:" + domain
	}

	// Set default and max values for total results requested
	totalRequested := params.Num
	if totalRequested <= 0 {
		totalRequested = ResultsPerPage // default to 10
	} else if totalRequested > MaxResultsPerRequest {
		totalRequested = MaxResultsPerRequest // cap at 100
	}

	// Calculate how many pages we need to fetch
	pagesNeeded := (totalRequested + ResultsPerPage - 1) / ResultsPerPage // ceiling division
	if pagesNeeded > MaxPagesToFetch {
		pagesNeeded = MaxPagesToFetch
	}

	// Initialize response
	result := &SearchResponse{
		OrganicResults: []OrganicResult{},
	}

	// Starting offset (considering user's Start parameter)
	currentStart := params.Start
	pagesFetched := 0

	// Fetch pages until we have enough results or no more pages available
	for pagesFetched < pagesNeeded && len(result.OrganicResults) < totalRequested {
		pageResults, pagination, err := h.fetchPage(query, canonicalLocation, params.Hl, params.Gl, currentStart)
		if err != nil {
			// If this is the first page, return the error
			// If we already have some results, return what we have
			if pagesFetched == 0 {
				return nil, err
			}
			break
		}

		pagesFetched++

		// Append results
		for _, res := range pageResults {
			if len(result.OrganicResults) >= totalRequested {
				break
			}
			// Update position to be sequential across all pages
			res.Position = len(result.OrganicResults) + 1
			result.OrganicResults = append(result.OrganicResults, res)
		}

		// Update pagination info (keep the last one)
		if pagination != nil {
			result.Pagination = *pagination
		}

		// Check if there are more pages available
		if pagination == nil || pagination.Next == "" {
			// No more pages available
			break
		}

		// No results returned means we've reached the end
		if len(pageResults) == 0 {
			break
		}

		// Move to next page
		currentStart += ResultsPerPage
	}

	// Update response metadata
	result.TotalResults = len(result.OrganicResults)
	result.PagesFetched = pagesFetched

	// If FirecrawlHandler is configured, scrape all organic result websites
	log.Printf("[GoogleSearchHandler] firecrawlHandler is nil: %v, organic results count: %d", h.firecrawlHandler == nil, len(result.OrganicResults))
	if h.firecrawlHandler != nil && len(result.OrganicResults) > 0 {
		log.Printf("[GoogleSearchHandler] Starting Firecrawl scraping for %d results", len(result.OrganicResults))
		scrapedMap := h.firecrawlHandler.ScrapeOrganicResults(result.OrganicResults)
		log.Printf("[GoogleSearchHandler] Firecrawl returned %d scraped pages", len(scrapedMap))

		// Enrich organic results with scraped content
		for i := range result.OrganicResults {
			link := result.OrganicResults[i].Link
			if scraped, exists := scrapedMap[link]; exists {
				if scraped.Success {
					log.Printf("[GoogleSearchHandler] Enriching result %d with scraped content (length: %d)", i+1, len(scraped.Markdown))
					result.OrganicResults[i].ScrapedContent = scraped.Markdown
				} else {
					log.Printf("[GoogleSearchHandler] Scrape failed for result %d: %s", i+1, scraped.Error)
					result.OrganicResults[i].ScrapeError = scraped.Error
				}
			}
		}
	} else {
		log.Printf("[GoogleSearchHandler] Skipping Firecrawl scraping (handler nil or no results)")
	}

	// If DataExtractorHandler is configured, extract company data from scraped content
	if h.dataExtractorHandler != nil && len(result.OrganicResults) > 0 {
		log.Printf("[GoogleSearchHandler] Starting data extraction for %d results", len(result.OrganicResults))
		ctx := context.Background()
		extractedMap := h.dataExtractorHandler.ExtractFromResults(ctx, result.OrganicResults)

		// Enrich organic results with extracted data
		for i := range result.OrganicResults {
			link := result.OrganicResults[i].Link
			if extracted, exists := extractedMap[link]; exists {
				result.OrganicResults[i].ExtractedData = extracted
			}
		}

		successCount := 0
		for _, extracted := range extractedMap {
			if extracted.Success {
				successCount++
			}
		}
		log.Printf("[GoogleSearchHandler] Data extraction complete: %d/%d successful", successCount, len(extractedMap))
	} else {
		log.Printf("[GoogleSearchHandler] Skipping data extraction (handler nil or no results)")
	}

	// If PreCallReportHandler is configured, generate pre-call reports
	log.Printf("[GoogleSearchHandler] preCallReportHandler is nil: %v, organic results count: %d", h.preCallReportHandler == nil, len(result.OrganicResults))
	if h.preCallReportHandler != nil && len(result.OrganicResults) > 0 {
		log.Printf("[GoogleSearchHandler] Starting pre-call report generation for %d results", len(result.OrganicResults))
		ctx := context.Background()
		reports := h.preCallReportHandler.GenerateReports(ctx, result.OrganicResults)

		// Enrich organic results with pre-call report (company_summary only)
		successCount := 0
		for i := range result.OrganicResults {
			link := result.OrganicResults[i].Link
			if report, exists := reports[link]; exists {
				if report.Success {
					result.OrganicResults[i].PreCallReport = report.CompanySummary
					successCount++
				}
			}
		}
		log.Printf("[GoogleSearchHandler] Pre-call report generation complete: %d/%d successful", successCount, len(reports))
	} else {
		log.Printf("[GoogleSearchHandler] Skipping pre-call report generation (handler nil or no results)")
	}

	// If ColdEmailHandler is configured, generate cold emails (after pre-call reports)
	log.Printf("[GoogleSearchHandler] coldEmailHandler is nil: %v, organic results count: %d", h.coldEmailHandler == nil, len(result.OrganicResults))
	if h.coldEmailHandler != nil && len(result.OrganicResults) > 0 {
		log.Printf("[GoogleSearchHandler] Starting cold email generation for %d results", len(result.OrganicResults))
		ctx := context.Background()

		// Build email generation inputs with pre-call report data
		var inputs []EmailGenerationInput
		for _, r := range result.OrganicResults {
			inputs = append(inputs, EmailGenerationInput{
				Result:        r,
				PreCallReport: r.PreCallReport, // Include pre-call report for better personalization
			})
		}

		emails := h.coldEmailHandler.GenerateEmails(ctx, inputs)

		// Enrich organic results with cold emails
		successCount := 0
		for i := range result.OrganicResults {
			link := result.OrganicResults[i].Link
			if email, exists := emails[link]; exists {
				if email.Success {
					result.OrganicResults[i].ColdEmail = email
					successCount++
				}
			}
		}
		log.Printf("[GoogleSearchHandler] Cold email generation complete: %d/%d successful", successCount, len(emails))
	} else {
		log.Printf("[GoogleSearchHandler] Skipping cold email generation (handler nil or no results)")
	}

	return result, nil
}

// Helper functions to safely extract values from map[string]interface{}
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0
}
