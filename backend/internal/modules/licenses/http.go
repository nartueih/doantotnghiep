package licenses

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
	routes := v1.Group("/licenses")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	routes.GET("", h.list)
	routes.GET("/:id/key", h.revealKey)
	routes.POST("", h.create)
	routes.PUT("/:id", h.update)
	routes.PATCH("/:id/archive", h.archive)
}

func (h *HTTPHandler) archive(c *gin.Context) {
	item, err := h.service.Archive(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeLicenseError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionArchive, audit.EntityLicense, item.ID, map[string]any{
		"name": item.Name, "software_product_id": item.SoftwareProductID,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) list(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeLicenseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) create(c *gin.Context) {
	input, ok := bindLicenseInput(c)
	if !ok {
		return
	}
	item, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeLicenseError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionCreate, audit.EntityLicense, item.ID, map[string]any{
		"name": item.Name, "software_product_id": item.SoftwareProductID, "seat_count": item.SeatCount,
		"key_configured": item.KeyHint != "",
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *HTTPHandler) update(c *gin.Context) {
	input, ok := bindLicenseInput(c)
	if !ok {
		return
	}
	item, err := h.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeLicenseError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionUpdate, audit.EntityLicense, item.ID, map[string]any{
		"name": item.Name, "software_product_id": item.SoftwareProductID, "seat_count": item.SeatCount,
		"key_changed": input.LicenseKey != "",
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) revealKey(c *gin.Context) {
	licenseKey, err := h.service.RevealKey(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeLicenseError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionViewKey, audit.EntityLicense, c.Param("id"), nil); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"license_key": licenseKey})
}

func bindLicenseInput(c *gin.Context) (Input, bool) {
	var request struct {
		SoftwareProductID string  `json:"software_product_id" binding:"required"`
		Name              string  `json:"name" binding:"required"`
		LicenseType       string  `json:"license_type" binding:"required"`
		AssignmentType    string  `json:"assignment_type" binding:"required"`
		SeatCount         int     `json:"seat_count" binding:"required"`
		LicenseKey        string  `json:"license_key"`
		Vendor            string  `json:"vendor"`
		PurchasedAt       string  `json:"purchased_at"`
		StartsAt          string  `json:"starts_at"`
		ExpiresAt         string  `json:"expires_at"`
		Cost              float64 `json:"cost"`
		Currency          string  `json:"currency"`
		Notes             string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "required license fields are missing"})
		return Input{}, false
	}
	return Input{
		SoftwareProductID: request.SoftwareProductID,
		Name:              request.Name,
		LicenseType:       request.LicenseType,
		AssignmentType:    request.AssignmentType,
		SeatCount:         request.SeatCount,
		LicenseKey:        request.LicenseKey,
		Vendor:            request.Vendor,
		PurchasedAt:       request.PurchasedAt,
		StartsAt:          request.StartsAt,
		ExpiresAt:         request.ExpiresAt,
		Cost:              request.Cost,
		Currency:          request.Currency,
		Notes:             request.Notes,
	}, true
}

func writeLicenseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidData),
		errors.Is(err, ErrInvalidType),
		errors.Is(err, ErrInvalidAssignment),
		errors.Is(err, ErrInvalidSeatCount),
		errors.Is(err, ErrInvalidDate),
		errors.Is(err, ErrExpirationRequired),
		errors.Is(err, ErrInvalidDateRange),
		errors.Is(err, ErrInvalidCost):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrSoftwareNotFound), errors.Is(err, ErrNotFound), errors.Is(err, ErrKeyNotSet):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrSeatCountBelowUsage),
		errors.Is(err, ErrArchived),
		errors.Is(err, ErrAlreadyArchived),
		errors.Is(err, ErrActiveAssignments):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
