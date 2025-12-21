package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service *Service
}

type libraryRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status" binding:"required"`
}

type progressRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter" binding:"gte=0"`
	Status         string `json:"status"`
}

// RegisterRoutes registers user library HTTP endpoints.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	h := &Handler{Service: svc}

	r.POST("/users/library", h.HandleAddToLibrary)
	r.GET("/users/library", h.HandleGetLibrary)
	r.PUT("/users/progress", h.HandleUpdateProgress)
}

// HandleAddToLibrary implements UC-005: add manga to library.
func (h *Handler) HandleAddToLibrary(c *gin.Context) {
	userID := c.GetString("user_id")
	var req libraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addReq := AddToLibraryRequest{
		MangaID:        req.MangaID,
		CurrentChapter: req.CurrentChapter,
		Status:         req.Status,
	}

	result, err := h.Service.AddToLibrary(userID, addReq)
	if err != nil {
		// UC-005 Alternative Flow A2: Database error - show retry option
		errorMsg := err.Error()
		if errorMsg == "database_error: failed to check library entry" || 
		   errorMsg == "database_error: failed to save library entry" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error. Please try again.",
				"type":  "database_error",
				"retry": true,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to add to library. Please try again.",
				"type":  "unknown_error",
				"retry": true,
			})
		}
		return
	}

	// UC-005 Main Success Scenario: Confirm addition and return status
	response := gin.H{
		"message": "Manga added to library successfully",
		"status":  result.Status,
	}
	
	// UC-005 Alternative Flow A1: Manga already in library
	if result.Status == "already_exists" {
		response["message"] = "Manga already in library. Progress updated successfully!"
	}

	c.JSON(http.StatusOK, response)
}

// HandleGetLibrary retrieves user's library.
func (h *Handler) HandleGetLibrary(c *gin.Context) {
	userID := c.GetString("user_id")
	items, err := h.Service.GetLibrary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query library"})
		return
	}

	c.JSON(http.StatusOK, items)
}

// HandleUpdateProgress updates user's reading progress.
func (h *Handler) HandleUpdateProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	var req progressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateReq := UpdateProgressRequest{
		MangaID:        req.MangaID,
		CurrentChapter: req.CurrentChapter,
		Status:         req.Status,
	}

	if err := h.Service.UpdateProgress(userID, updateReq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}
