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

	err := h.Service.AddToLibrary(userID, addReq)
	if err != nil {
		switch err.Error() {
		case "manga_not_found":
			c.JSON(http.StatusNotFound, gin.H{"error": "manga not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save library entry"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "saved"})
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

