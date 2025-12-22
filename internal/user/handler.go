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

type notifyRequest struct {
	MangaID string `json:"manga_id" binding:"required"`
}

// RegisterRoutes registers user library HTTP endpoints.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	h := &Handler{Service: svc}

	r.POST("/users/library", h.HandleAddToLibrary)
	r.GET("/users/library", h.HandleGetLibrary)
	r.PUT("/users/progress", h.HandleUpdateProgress)
	r.POST("/users/notifications", h.HandleSubscribeNotifications)
	r.GET("/users/notifications/:manga_id", h.HandleCheckNotificationSubscription)
	r.DELETE("/users/notifications/:manga_id", h.HandleUnsubscribeNotifications)
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

// HandleUpdateProgress implements UC-006: update reading progress.
func (h *Handler) HandleUpdateProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req progressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
			"type":  "validation_error",
		})
		return
	}

	updateReq := UpdateProgressRequest{
		MangaID:        req.MangaID,
		CurrentChapter: req.CurrentChapter,
		Status:         req.Status,
	}

	result, err := h.Service.UpdateProgress(userID, updateReq)
	if err != nil {
		errorMsg := err.Error()
		
		// UC-006 Alternative Flow A1: Invalid chapter number - show validation error
		if errorMsg == "validation_error: chapter number cannot be negative" ||
			errorMsg == "validation_error: chapter number exceeds total chapters" ||
			errorMsg == "validation_error: manga is not in user's library" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": errorMsg,
				"type":  "validation_error",
			})
			return
		}

		// Database errors
		if errorMsg == "database_error: failed to check library entry" ||
			errorMsg == "database_error: failed to update progress" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error. Please try again.",
				"type":  "database_error",
				"retry": true,
			})
			return
		}

		// Unknown errors
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update progress: " + errorMsg,
			"type":  "unknown_error",
			"retry": true,
		})
		return
	}

	// UC-006 Main Success Scenario Step 5: Confirm update to user
	response := gin.H{
		"message":        "Progress updated successfully",
		"broadcast_sent": result.BroadcastSent,
	}

	// UC-006 Alternative Flow A2: TCP server unavailable - inform user but confirm local update
	if !result.BroadcastSent && result.BroadcastError != "" {
		response["broadcast_error"] = "Progress updated locally, but broadcast failed (will be queued)"
		response["warning"] = "TCP server unavailable. Progress saved locally."
	}

	c.JSON(http.StatusOK, response)
}

// HandleSubscribeNotifications subscribes the user to notifications for a manga (UC-009).
func (h *Handler) HandleSubscribeNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req notifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
			"type":  "validation_error",
		})
		return
	}

	if err := h.Service.SubscribeToMangaNotifications(userID, req.MangaID); err != nil {
		errorMsg := err.Error()
		if errorMsg == "database_error: failed to save notification subscription" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error. Please try again.",
				"type":  "database_error",
				"retry": true,
			})
			return
		}
		if errorMsg == "validation_error: user_id and manga_id are required" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": errorMsg,
				"type":  "validation_error",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to subscribe to notifications: " + errorMsg,
			"type":  "unknown_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscribed to notifications for this manga",
	})
}

// HandleCheckNotificationSubscription returns whether the user is subscribed for a given manga.
func (h *Handler) HandleCheckNotificationSubscription(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	mangaID := c.Param("manga_id")
	subscribed, err := h.Service.IsSubscribedToMangaNotifications(userID, mangaID)
	if err != nil {
		errorMsg := err.Error()
		if errorMsg == "database_error: failed to check notification subscription" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error. Please try again.",
				"type":  "database_error",
			})
			return
		}
		if errorMsg == "validation_error: user_id and manga_id are required" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": errorMsg,
				"type":  "validation_error",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check subscription: " + errorMsg,
			"type":  "unknown_error",
		})
		return
	}

	// Best-effort re-register on UDP if subscribed
	if subscribed {
		go h.Service.ReRegisterNotification(userID, mangaID)
	}

	c.JSON(http.StatusOK, gin.H{
		"subscribed": subscribed,
	})
}

// HandleUnsubscribeNotifications disables notifications for a manga for the user.
func (h *Handler) HandleUnsubscribeNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	mangaID := c.Param("manga_id")
	if err := h.Service.UnsubscribeFromMangaNotifications(userID, mangaID); err != nil {
		errorMsg := err.Error()
		if errorMsg == "database_error: failed to delete notification subscription" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error. Please try again.",
				"type":  "database_error",
			})
			return
		}
		if errorMsg == "validation_error: user_id and manga_id are required" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": errorMsg,
				"type":  "validation_error",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to unsubscribe: " + errorMsg,
			"type":  "unknown_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notifications disabled for this manga",
	})
}
