package audit

import (
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
	routes := v1.Group("/audit-logs")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	routes.GET("", h.list)
}

func (h *HTTPHandler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.service.List(c.Request.Context(), Filter{
		Action:     c.Query("action"),
		EntityType: c.Query("entity_type"),
		ActorID:    c.Query("actor_id"),
		Limit:      limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func RecordRequest(c *gin.Context, recorder Recorder, action, entityType, entityID string, metadata map[string]any) error {
	if recorder == nil {
		return nil
	}
	_, err := recorder.Record(c.Request.Context(), RecordInput{
		ActorID:    auth.CurrentUserID(c),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Metadata:   metadata,
		IPAddress:  c.ClientIP(),
	})
	return err
}

func WriteError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "operation completed but audit log could not be recorded"})
}
