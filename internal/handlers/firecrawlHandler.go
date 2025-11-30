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
	// DefaultScrapeTimeout is the timeout for scraping a single URL
	DefaultScrapeTimeout = 30 * time.Second
	// MaxConcurrentScrapes limits how many URLs we scrape in parallel
	MaxConcurrentScrapes = 5
)

// ScrapedPage represents the scraped content from a single URL
type ScrapedPage struct {
	// URL that was scraped
	URL string `json:"url"`
	// Markdown content extracted from the page
	Markdown string `json:"markdown,omitempty"`
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

// ScrapeURL scrapes a single URL and returns its markdown content
func (h *FirecrawlHandler) ScrapeURL(targetURL string) (*ScrapedPage, error) {
	log.Printf("[FirecrawlHandler] ScrapeURL called for: %s", targetURL)
	result := &ScrapedPage{
		URL:     targetURL,
		Success: false,
	}

	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil || parsedURL.Host == "" {
		log.Printf("[FirecrawlHandler] Invalid URL: %s", targetURL)
		result.Error = "invalid URL"
		return result, nil
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
		scrapedData, err := h.app.ScrapeURL(targetURL, nil)
		resultChan <- scrapeResult{data: scrapedData, err: err}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		log.Printf("[FirecrawlHandler] Timeout exceeded for: %s", targetURL)
		result.Error = "scrape timeout exceeded"
		return result, nil
	case res := <-resultChan:
		if res.err != nil {
			log.Printf("[FirecrawlHandler] Scrape error for %s: %v", targetURL, res.err)
			result.Error = res.err.Error()
			return result, nil
		}
		if res.data != nil {
			log.Printf("[FirecrawlHandler] Successfully scraped %s (markdown length: %d)", targetURL, len(res.data.Markdown))
			result.Markdown = res.data.Markdown
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
// Returns a map of URL -> ScrapedPage for easy lookup
func (h *FirecrawlHandler) ScrapeOrganicResults(organicResults []OrganicResult) map[string]ScrapedPage {
	log.Printf("[FirecrawlHandler] ScrapeOrganicResults called with %d results", len(organicResults))
	if len(organicResults) == 0 {
		log.Printf("[FirecrawlHandler] No organic results to scrape")
		return make(map[string]ScrapedPage)
	}

	// Extract unique URLs from organic results
	urlSet := make(map[string]struct{})
	var urls []string
	for _, result := range organicResults {
		if result.Link != "" {
			if _, exists := urlSet[result.Link]; !exists {
				urlSet[result.Link] = struct{}{}
				urls = append(urls, result.Link)
			}
		}
	}

	log.Printf("[FirecrawlHandler] Scraping %d unique URLs", len(urls))

	// Scrape all URLs
	scrapedPages := h.ScrapeURLs(urls)

	// Build result map
	resultMap := make(map[string]ScrapedPage, len(scrapedPages))
	successCount := 0
	for _, page := range scrapedPages {
		resultMap[page.URL] = page
		if page.Success {
			successCount++
		}
	}

	log.Printf("[FirecrawlHandler] Scraping complete: %d/%d successful", successCount, len(urls))
	return resultMap
}
