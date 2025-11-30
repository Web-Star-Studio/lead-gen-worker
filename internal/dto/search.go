package dto

// SearchRequest represents the incoming search request body
// @Description Search request parameters for Google search
type SearchRequest struct {
	// Search query string
	Q string `json:"q" binding:"required" example:"escritório de contabilidade"`
	// Location for geo-targeted search
	Location string `json:"location" binding:"required" example:"Recife"`
	// Language code for search results
	Hl string `json:"hl" example:"pt-br"`
	// Country code for search results
	Gl string `json:"gl" example:"br"`
	// List of domains to exclude from search results
	ExcludeDomains []string `json:"exclude_domains" example:"instagram.com,linkedin.com,facebook.com"`
	// Total number of results to return (default: 10, max: 100). Multiple pages will be fetched automatically if needed.
	Num int `json:"num" example:"50"`
	// Result offset for pagination (default: 0)
	Start int `json:"start" example:"0"`
}

// ErrorResponse represents an error response
// @Description Error response returned when request fails
type ErrorResponse struct {
	// Error message describing what went wrong
	Error string `json:"error" example:"Key: 'SearchRequest.Q' Error:Field validation for 'Q' failed on the 'required' tag"`
}
