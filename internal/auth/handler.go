package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service   *Service
	JWTSecret []byte
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRoutes wires the auth HTTP endpoints and returns a JWT middleware
// that can be used to protect other routes.
func RegisterRoutes(r *gin.Engine, svc *Service, jwtSecret []byte) gin.HandlerFunc {
	h := &Handler{
		Service:   svc,
		JWTSecret: jwtSecret,
	}

	r.POST("/auth/register", h.HandleRegister)
	r.POST("/auth/login", h.HandleLogin)

	return h.JWTMiddleware
}

// HandleRegister implements UC-001 over HTTP.
func (h *Handler) HandleRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "Password' failed on the 'min'") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "password must be at least 8 characters long",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registration data"})
		return
	}

	if err := h.Service.RegisterUser(req.Username, req.Email, req.Password); err != nil {
		switch err.Error() {
		case "weak_password":
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "password too weak",
				"rules": "minimum 8 characters, must contain both letters and numbers",
			})
		case "username_exists":
			c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		case "email_exists":
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

// HandleLogin implements UC-002 over HTTP.
func (h *Handler) HandleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Username == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username or email is required"})
		return
	}

	byEmail := req.Email != ""
	identifier := req.Username
	if byEmail {
		identifier = req.Email
	}

	user, err := h.Service.AuthenticateUser(identifier, req.Password, byEmail)
	if err != nil {
		switch err.Error() {
		case "account_not_found":
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "account not found",
				"message": "please register before logging in",
			})
		case "invalid_credentials":
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
		}
		return
	}

	token, err := GenerateJWT(h.JWTSecret, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// JWTMiddleware validates JWTs and injects user_id into the Gin context.
func (h *Handler) JWTMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	raw := authHeader[7:]
	userID, err := ParseUserIDFromToken(h.JWTSecret, raw)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	c.Set("user_id", userID)
	c.Next()
}


