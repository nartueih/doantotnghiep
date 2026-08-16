package selfservice

import (
	"net/http"

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
	routes := v1.Group("/me")
	routes.Use(h.auth.Authenticate())
	routes.GET("/devices", h.devices)
	routes.GET("/licenses", h.licenses)
}

func (h *HTTPHandler) devices(c *gin.Context) {
	items, err := h.service.Devices(c.Request.Context(), auth.CurrentUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) licenses(c *gin.Context) {
	items, err := h.service.Licenses(c.Request.Context(), auth.CurrentUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}
