package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const userIDContextKey = "authenticated_user_id"

type HTTPHandler struct {
	service *Service
	tokens  *TokenManager
}

func NewHTTPHandler(service *Service, tokens *TokenManager) *HTTPHandler {
	return &HTTPHandler{service: service, tokens: tokens}
}

func (h *HTTPHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	routes := v1.Group("/auth")
	routes.POST("/login", h.login)
	routes.POST("/refresh", h.refresh)
	routes.POST("/logout", h.logout)
	routes.GET("/me", h.authenticate(), h.me)
}

func (h *HTTPHandler) login(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	result, err := h.service.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) refresh(c *gin.Context) {
	refreshToken, ok := bindRefreshToken(c)
	if !ok {
		return
	}

	result, err := h.service.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) logout(c *gin.Context) {
	refreshToken, ok := bindRefreshToken(c)
	if !ok {
		return
	}

	if err := h.service.Logout(c.Request.Context(), refreshToken); err != nil {
		writeAuthError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *HTTPHandler) me(c *gin.Context) {
	userID, _ := c.Get(userIDContextKey)
	user, err := h.service.Me(c.Request.Context(), userID.(string))
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *HTTPHandler) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		scheme, rawToken, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(rawToken) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "a valid bearer token is required"})
			return
		}

		claims, err := h.tokens.ParseAccess(strings.TrimSpace(rawToken))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "access token is invalid or expired"})
			return
		}
		c.Set(userIDContextKey, claims.Subject)
		c.Next()
	}
}

func bindRefreshToken(c *gin.Context) (string, bool) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return "", false
	}
	return request.RefreshToken, true
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email or password is incorrect"})
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrInvalidRefreshToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token is invalid or expired"})
	case errors.Is(err, ErrAccountLocked):
		c.JSON(http.StatusForbidden, gin.H{"error": "account is locked"})
	case errors.Is(err, ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
