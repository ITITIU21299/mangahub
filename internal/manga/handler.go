package manga

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service *Service
}

// RegisterRoutes registers manga HTTP endpoints.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	h := &Handler{Service: svc}

	r.GET("/manga", h.HandleListManga)
	r.GET("/manga/:id", h.HandleGetManga)
}

func (h *Handler) HandleListManga(c *gin.Context) {
	// Parse query parameters
	query := c.Query("q")
	genre := c.Query("genre")
	status := c.Query("status")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	params := SearchParams{
		Query:  query,
		Genre:  genre,
		Status: status,
		Page:   page,
		Limit:  limit,
	}

	result, err := h.Service.SearchManga(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query manga"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Data,
		"pagination": gin.H{
			"page":        result.Page,
			"limit":       result.Limit,
			"total":       result.Total,
			"total_pages": result.TotalPages,
		},
	})
}

// HandleGetManga retrieves a single manga by ID, optionally including user progress if logged in.
func (h *Handler) HandleGetManga(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id") // Will be empty if not authenticated

	result, err := h.Service.GetMangaByIDWithProgress(id, userID)
	if err != nil {
		if err.Error() == "not_found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query manga"})
		return
	}

	// Return manga with optional user progress
	response := gin.H{
		"id":             result.Manga.ID,
		"title":          result.Manga.Title,
		"author":         result.Manga.Author,
		"genres":         result.Manga.Genres,
		"status":         result.Manga.Status,
		"total_chapters": result.Manga.TotalChapters,
		"description":    result.Manga.Description,
		"cover_url":      result.Manga.CoverURL,
		"user_progress":  result.UserProgress,
	}

	c.JSON(http.StatusOK, response)
}
