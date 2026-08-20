package selfservice

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/licenses"
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
	routes := v1.Group("/me")
	routes.Use(h.auth.Authenticate())
	routes.GET("/devices", h.devices)
	routes.GET("/licenses", h.licenses)
	routes.GET("/licenses/:assignment_id/key", h.revealLicenseKey)
}

func (h *HTTPHandler) revealLicenseKey(c *gin.Context) {
	access, err := h.service.RevealLicenseKey(c.Request.Context(), auth.CurrentUserID(c), c.Param("assignment_id"))
	if err != nil {
		switch {
		case errors.Is(err, ErrAssignmentNotFound), errors.Is(err, licenses.ErrNotFound), errors.Is(err, licenses.ErrKeyNotSet):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, licenses.ErrEmployeeKeyNotAllowed):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, licenses.ErrKeyUnavailable):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionViewKey, audit.EntityLicense, access.LicenseID, map[string]any{
		"assignment_id": access.AssignmentID,
		"access_scope":  "employee_self_service",
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, access)
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
