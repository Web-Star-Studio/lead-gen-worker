package handlers

import (
	"context"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/mendableai/firecrawl-go/v2"
)

const (
	// DefaultScrapeTimeout is the timeout for scraping a single URL (increased for JS-heavy sites)
	DefaultScrapeTimeout = 45 * time.Second
	// MaxConcurrentScrapes limits how many URLs we scrape in parallel
	MaxConcurrentScrapes = 5
	// DefaultWaitFor is the time to wait for JavaScript to load (ms)
	DefaultWaitFor = 2000
	// DefaultFirecrawlTimeout is the timeout sent to Firecrawl API (ms)
	DefaultFirecrawlTimeout = 30000
)

// ScrapedPage represents the scraped content from a single URL
type ScrapedPage struct {
	// URL that was scraped
	URL string `json:"url"`
	// Markdown content extracted from the page
	Markdown string `json:"markdown,omitempty"`
	// Links found on the page (useful for finding contact pages, social media, etc.)
	Links []string `json:"links,omitempty"`
	// Error message if scraping failed
	Error string `json:"error,omitempty"`
	// Success indicates whether the scrape was successful
	Success bool `json:"success"`
}

// FirecrawlHandler handles website scraping using Firecrawl API
type FirecrawlHandler struct {
	app     *firecrawl.FirecrawlApp
	timeout time.Duration
}

// NewFirecrawlHandler creates a new FirecrawlHandler instance
// apiKey is required, apiURL can be empty to use the default Firecrawl API
func NewFirecrawlHandler(apiKey string, apiURL string) (*FirecrawlHandler, error) {
	log.Printf("[FirecrawlHandler] Initializing with apiURL: %q", apiURL)
	app, err := firecrawl.NewFirecrawlApp(apiKey, apiURL)
	if err != nil {
		log.Printf("[FirecrawlHandler] Failed to create FirecrawlApp: %v", err)
		return nil, err
	}

	log.Printf("[FirecrawlHandler] Successfully created FirecrawlApp")
	return &FirecrawlHandler{
		app:     app,
		timeout: DefaultScrapeTimeout,
	}, nil
}

// SetTimeout allows customizing the scrape timeout
func (h *FirecrawlHandler) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
}

// normalizeToRootURL extracts the root URL (scheme + host) from any URL
// e.g., "https://example.com/support/contact" -> "https://example.com"
func normalizeToRootURL(targetURL string) (string, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}
	if parsedURL.Host == "" {
		return "", nil
	}
	// Reconstruct URL with only scheme and host (root domain)
	rootURL := &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
	}
	return rootURL.String(), nil
}

// ScrapeURL scrapes a single URL and returns its markdown content
// Note: URLs are normalized to root domain (e.g., https://example.com/page -> https://example.com)
func (h *FirecrawlHandler) ScrapeURL(targetURL string) (*ScrapedPage, error) {
	log.Printf("[FirecrawlHandler] ScrapeURL called for: %s", targetURL)

	// Normalize URL to root domain
	normalizedURL, err := normalizeToRootURL(targetURL)
	if err != nil || normalizedURL == "" {
		log.Printf("[FirecrawlHandler] Invalid URL: %s", targetURL)
		return &ScrapedPage{
			URL:     targetURL,
			Error:   "invalid URL",
			Success: false,
		}, nil
	}

	// Log if URL was normalized
	if normalizedURL != targetURL {
		log.Printf("[FirecrawlHandler] URL normalized: %s -> %s", targetURL, normalizedURL)
	}

	result := &ScrapedPage{
		URL:     normalizedURL, // Store the normalized URL
		Success: false,
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Channel to receive scrape result
	type scrapeResult struct {
		data *firecrawl.FirecrawlDocument
		err  error
	}
	resultChan := make(chan scrapeResult, 1)

	// Perform scrape in goroutine to support timeout
	go func() {
		// Configure scrape parameters for robust lead generation scraping
		onlyMainContent := false // Include header/footer for contact info (phone, email, address)
		waitFor := DefaultWaitFor
		timeout := DefaultFirecrawlTimeout
		maxAge := 0 // Force fresh scrape - lead data should be current

		scrapeParams := &firecrawl.ScrapeParams{
			Formats:         []string{"markdown", "links"}, // Include links for better data extraction
			OnlyMainContent: &onlyMainContent,
			WaitFor:         &waitFor, // Wait for JavaScript to load (SPAs, React sites)
			Timeout:         &timeout, // Firecrawl API timeout
			MaxAge:          &maxAge,  // Always fetch fresh content, don't use cache
		}
		scrapedData, err := h.app.ScrapeURL(normalizedURL, scrapeParams)
		resultChan <- scrapeResult{data: scrapedData, err: err}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		log.Printf("[FirecrawlHandler] Timeout exceeded for: %s", normalizedURL)
		result.Error = "scrape timeout exceeded"
		return result, nil
	case res := <-resultChan:
		if res.err != nil {
			log.Printf("[FirecrawlHandler] Scrape error for %s: %v", normalizedURL, res.err)
			result.Error = res.err.Error()
			return result, nil
		}
		if res.data != nil {
			log.Printf("[FirecrawlHandler] Successfully scraped %s (markdown: %d chars, links: %d)",
				normalizedURL, len(res.data.Markdown), len(res.data.Links))
			result.Markdown = res.data.Markdown
			result.Links = res.data.Links
			result.Success = true
		}
	}

	return result, nil
}

// ScrapeURLs scrapes multiple URLs concurrently and returns their markdown content
// It uses a semaphore pattern to limit concurrent scrapes
func (h *FirecrawlHandler) ScrapeURLs(urls []string) []ScrapedPage {
	if len(urls) == 0 {
		return []ScrapedPage{}
	}

	results := make([]ScrapedPage, len(urls))
	var wg sync.WaitGroup

	// Semaphore to limit concurrent scrapes
	semaphore := make(chan struct{}, MaxConcurrentScrapes)

	for i, targetURL := range urls {
		wg.Add(1)
		go func(index int, u string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			scraped, _ := h.ScrapeURL(u)
			if scraped != nil {
				results[index] = *scraped
			} else {
				results[index] = ScrapedPage{
					URL:     u,
					Success: false,
					Error:   "unknown error",
				}
			}
		}(i, targetURL)
	}

	wg.Wait()
	return results
}

// ScrapeOrganicResults takes organic search results and scrapes each website's homepage
// Returns a map of original URL -> ScrapedPage for easy lookup
// Note: URLs are normalized to root domain before scraping, but the map is keyed by original URL
func (h *FirecrawlHandler) ScrapeOrganicResults(organicResults []OrganicResult) map[string]ScrapedPage {
	log.Printf("[FirecrawlHandler] ScrapeOrganicResults called with %d results", len(organicResults))
	if len(organicResults) == 0 {
		log.Printf("[FirecrawlHandler] No organic results to scrape")
		return make(map[string]ScrapedPage)
	}

	// Extract unique URLs and track original -> normalized mapping
	// Multiple original URLs may map to the same normalized (root) URL
	urlSet := make(map[string]struct{})
	normalizedToOriginals := make(map[string][]string) // normalized URL -> list of original URLs
	var urls []string

	for _, result := range organicResults {
		if result.Link != "" {
			if _, exists := urlSet[result.Link]; !exists {
				urlSet[result.Link] = struct{}{}
				urls = append(urls, result.Link)

				// Track the mapping from normalized to original URLs
				normalized, err := normalizeToRootURL(result.Link)
				if err == nil && normalized != "" {
					normalizedToOriginals[normalized] = append(normalizedToOriginals[normalized], result.Link)
				}
			}
		}
	}

	log.Printf("[FirecrawlHandler] Scraping %d unique URLs (may dedupe to fewer root domains)", len(urls))

	// Scrape all URLs (they will be normalized internally)
	scrapedPages := h.ScrapeURLs(urls)

	// Build result map keyed by ORIGINAL URLs (not normalized)
	// This ensures the lookup in googleSearchHandler works correctly
	resultMap := make(map[string]ScrapedPage, len(urls))
	successCount := 0

	for _, page := range scrapedPages {
		// page.URL is the normalized URL, find all original URLs that map to it
		if originals, exists := normalizedToOriginals[page.URL]; exists {
			for _, originalURL := range originals {
				resultMap[originalURL] = page
			}
		}
		// Also add by normalized URL as fallback
		resultMap[page.URL] = page

		if page.Success {
			successCount++
		}
	}

	log.Printf("[FirecrawlHandler] Scraping complete: %d/%d successful", successCount, len(scrapedPages))
	return resultMap
}
