package assignments

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
)

type HTTPHandler struct {
	service *Service
	auth    *auth.HTTPHandler
	audit   audit.Recorder
}

func NewHTTPHandler(service *Service, authHandler *auth.HTTPHandler, recorders ...audit.Recorder) *HTTPHandler {
	handler := &HTTPHandler{service: service, auth: authHandler}
	if len(recorders) > 0 {
		handler.audit = recorders[0]
	}
	return handler
}

func (h *HTTPHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	routes := v1.Group("/license-assignments")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	routes.GET("", h.list)
	routes.POST("", h.create)
	routes.PATCH("/:id/revoke", h.revoke)
}

func (h *HTTPHandler) list(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeAssignmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) create(c *gin.Context) {
	var request struct {
		LicenseID string `json:"license_id" binding:"required"`
		UserID    string `json:"user_id"`
		DeviceID  string `json:"device_id"`
		Notes     string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_id and exactly one assignment target are required"})
		return
	}
	item, err := h.service.Create(c.Request.Context(), auth.CurrentUserID(c), CreateInput{
		LicenseID: request.LicenseID,
		UserID:    request.UserID,
		DeviceID:  request.DeviceID,
		Notes:     request.Notes,
	})
	if err != nil {
		writeAssignmentError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionAssign, audit.EntityAssignment, item.ID, map[string]any{
		"license_id": item.LicenseID, "user_id": item.UserID, "device_id": item.DeviceID,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *HTTPHandler) revoke(c *gin.Context) {
	item, err := h.service.Revoke(c.Request.Context(), auth.CurrentUserID(c), c.Param("id"))
	if err != nil {
		writeAssignmentError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionRevoke, audit.EntityAssignment, item.ID, map[string]any{
		"license_id": item.LicenseID,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func writeAssignmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidTarget):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrLicenseNotFound), errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAssignmentType), errors.Is(err, ErrTargetUnavailable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, ErrLicenseInactive), errors.Is(err, ErrDuplicate), errors.Is(err, ErrNoAvailableSeats):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
