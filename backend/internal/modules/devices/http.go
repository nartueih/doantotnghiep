package devices

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
	routes := v1.Group("/devices")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	routes.GET("", h.list)
	routes.POST("", h.create)
	routes.PUT("/:id", h.update)
	routes.PATCH("/:id/status", h.changeStatus)
	routes.PATCH("/:id/assignment", h.assign)
}

func (h *HTTPHandler) list(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeDeviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) create(c *gin.Context) {
	input, ok := bindDeviceInput(c)
	if !ok {
		return
	}
	item, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeDeviceError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionCreate, audit.EntityDevice, item.ID, map[string]any{
		"asset_code": item.AssetCode, "device_type": item.DeviceType,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *HTTPHandler) update(c *gin.Context) {
	input, ok := bindDeviceInput(c)
	if !ok {
		return
	}
	item, err := h.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeDeviceError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionUpdate, audit.EntityDevice, item.ID, map[string]any{
		"asset_code": item.AssetCode, "device_type": item.DeviceType,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) changeStatus(c *gin.Context) {
	var request struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	item, err := h.service.ChangeStatus(c.Request.Context(), c.Param("id"), request.Status)
	if err != nil {
		writeDeviceError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionStatusChange, audit.EntityDevice, item.ID, map[string]any{
		"status": item.Status,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) assign(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body must be valid JSON"})
		return
	}
	item, err := h.service.Assign(c.Request.Context(), c.Param("id"), request.UserID)
	if err != nil {
		writeDeviceError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionAssign, audit.EntityDevice, item.ID, map[string]any{
		"assigned_user_id": item.AssignedUserID,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func bindDeviceInput(c *gin.Context) (Input, bool) {
	var request struct {
		AssetCode         string `json:"asset_code" binding:"required"`
		SerialNumber      string `json:"serial_number"`
		Name              string `json:"name" binding:"required"`
		DeviceType        string `json:"device_type" binding:"required"`
		Manufacturer      string `json:"manufacturer"`
		Model             string `json:"model"`
		PurchasedAt       string `json:"purchased_at"`
		WarrantyExpiresAt string `json:"warranty_expires_at"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "asset_code, name and device_type are required"})
		return Input{}, false
	}
	return Input{
		AssetCode:         request.AssetCode,
		SerialNumber:      request.SerialNumber,
		Name:              request.Name,
		DeviceType:        request.DeviceType,
		Manufacturer:      request.Manufacturer,
		Model:             request.Model,
		PurchasedAt:       request.PurchasedAt,
		WarrantyExpiresAt: request.WarrantyExpiresAt,
	}, true
}

func writeDeviceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidData),
		errors.Is(err, ErrInvalidDate),
		errors.Is(err, ErrInvalidDateRange),
		errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAssetCodeExists),
		errors.Is(err, ErrSerialExists),
		errors.Is(err, ErrDeviceAssigned),
		errors.Is(err, ErrDeviceUnavailable):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrUserUnavailable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
