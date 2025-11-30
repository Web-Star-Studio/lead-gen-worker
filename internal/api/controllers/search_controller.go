package controllers

import (
	"net/http"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SearchController handles search-related HTTP requests
type SearchController struct {
	searchHandler *handlers.GoogleSearchHandler
}

// NewSearchController creates a new SearchController instance
func NewSearchController(handler *handlers.GoogleSearchHandler) *SearchController {
	return &SearchController{
		searchHandler: handler,
	}
}

// Search godoc
// @Summary      Search Google for leads
// @Description  Perform a Google search using SerpAPI and retrieve organic results with advanced filtering options
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        request body dto.SearchRequest true "Search parameters"
// @Success      200 {object} handlers.SearchResponse "Successful search results"
// @Failure      400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Router       /search [post]
func (ctrl *SearchController) Search(c *gin.Context) {
	var req dto.SearchRequest

	// Bind and validate JSON request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	// Convert DTO to handler params
	params := handlers.GoogleSearchParams{
		Q:              req.Q,
		Location:       req.Location,
		Hl:             req.Hl,
		Gl:             req.Gl,
		ExcludeDomains: req.ExcludeDomains,
		Num:            req.Num,
		Start:          req.Start,
	}

	// Call the search handler
	result, err := ctrl.searchHandler.Search(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	// Return the search results
	c.JSON(http.StatusOK, result)
}
