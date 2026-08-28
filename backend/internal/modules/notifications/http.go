package notifications

import (
	"errors"
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
	routes := v1.Group("/me/notifications")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleEmployee))
	routes.GET("", h.list)
	routes.PATCH("/read-all", h.markAllRead)
	routes.PATCH("/:id/read", h.markRead)
}

func (h *HTTPHandler) list(c *gin.Context) {
	result, err := h.service.List(c.Request.Context(), auth.CurrentUserID(c))
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) markRead(c *gin.Context) {
	item, err := h.service.MarkRead(c.Request.Context(), auth.CurrentUserID(c), c.Param("id"))
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) markAllRead(c *gin.Context) {
	updated, err := h.service.MarkAllRead(c.Request.Context(), auth.CurrentUserID(c))
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

func writeNotificationError(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
