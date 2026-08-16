package dashboard

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
)

type HTTPHandler struct {
	service *Service
	auth    *auth.HTTPHandler
}

func NewHTTPHandler(service *Service, authHandler *auth.HTTPHandler) *HTTPHandler {
	return &HTTPHandler{service: service, auth: authHandler}
}

func (h *HTTPHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	routes := v1.Group("/dashboard")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	routes.GET("/summary", h.summary)
	routes.GET("/license-alerts", h.licenseAlerts)
}

func (h *HTTPHandler) summary(c *gin.Context) {
	result, err := h.service.Summary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) licenseAlerts(c *gin.Context) {
	days := 30
	if rawDays := c.Query("days"); rawDays != "" {
		parsed, err := strconv.Atoi(rawDays)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidExpiryWindow.Error()})
			return
		}
		days = parsed
	}
	items, err := h.service.LicenseAlerts(c.Request.Context(), days)
	if errors.Is(err, ErrInvalidExpiryWindow) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items), "expiry_window_days": days})
}
