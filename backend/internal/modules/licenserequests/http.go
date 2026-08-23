package licenserequests

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/assignments"
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
	employee := v1.Group("/me")
	employee.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleEmployee))
	employee.GET("/requestable-software", h.requestableSoftware)
	employee.GET("/license-requests", h.listMine)
	employee.POST("/license-requests", h.create)
	employee.PATCH("/license-requests/:id/cancel", h.cancel)

	admin := v1.Group("/license-requests")
	admin.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	admin.GET("", h.listAdmin)
	admin.PATCH("/:id/approve", h.approve)
	admin.PATCH("/:id/reject", h.reject)
}

func (h *HTTPHandler) requestableSoftware(c *gin.Context) {
	items, err := h.service.RequestableSoftware(c.Request.Context())
	if err != nil {
		writeRequestError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) listMine(c *gin.Context) {
	items, err := h.service.ListMine(c.Request.Context(), auth.CurrentUserID(c))
	if err != nil {
		writeRequestError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) create(c *gin.Context) {
	var request struct {
		SoftwareProductID string `json:"software_product_id" binding:"required"`
		Priority          string `json:"priority" binding:"required"`
		Reason            string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidData.Error()})
		return
	}
	item, err := h.service.Create(c.Request.Context(), auth.CurrentUserID(c), CreateInput{
		SoftwareProductID: request.SoftwareProductID,
		Priority:          request.Priority,
		Reason:            request.Reason,
	})
	if err != nil {
		writeRequestError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionRequest, audit.EntityLicenseRequest, item.ID, map[string]any{
		"software_product_id": item.SoftwareProductID,
		"priority":            item.Priority,
		"status":              item.Status,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *HTTPHandler) cancel(c *gin.Context) {
	item, err := h.service.Cancel(c.Request.Context(), auth.CurrentUserID(c), c.Param("id"))
	if err != nil {
		writeRequestError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionCancel, audit.EntityLicenseRequest, item.ID, map[string]any{
		"software_product_id": item.SoftwareProductID,
		"status":              item.Status,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) listAdmin(c *gin.Context) {
	items, err := h.service.ListAdmin(c.Request.Context(), Filter{
		Status: c.Query("status"), Priority: c.Query("priority"), Search: c.Query("search"),
	})
	if err != nil {
		writeRequestError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) approve(c *gin.Context) {
	var request struct {
		LicenseID    string `json:"license_id" binding:"required"`
		ResponseNote string `json:"response_note"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_id is required"})
		return
	}
	item, err := h.service.Approve(c.Request.Context(), auth.CurrentUserID(c), c.Param("id"), ApproveInput{
		LicenseID: request.LicenseID, ResponseNote: request.ResponseNote,
	})
	if err != nil {
		writeRequestError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionApprove, audit.EntityLicenseRequest, item.ID, map[string]any{
		"license_id": item.SelectedLicenseID, "assignment_id": item.AssignmentID, "status": item.Status,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) reject(c *gin.Context) {
	var request struct {
		DecisionReason string `json:"decision_reason" binding:"required"`
		ResponseNote   string `json:"response_note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision_reason and response_note are required"})
		return
	}
	item, err := h.service.Reject(c.Request.Context(), auth.CurrentUserID(c), c.Param("id"), RejectInput{
		DecisionReason: request.DecisionReason, ResponseNote: request.ResponseNote,
	})
	if err != nil {
		writeRequestError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionReject, audit.EntityLicenseRequest, item.ID, map[string]any{
		"decision_reason": item.DecisionReason, "status": item.Status,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func writeRequestError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidData), errors.Is(err, ErrInvalidPriority), errors.Is(err, ErrInvalidDecision):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrReviewerUnavailable):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrSoftwareNotFound), errors.Is(err, ErrLicenseNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrPendingDuplicate), errors.Is(err, ErrInvalidState), errors.Is(err, assignments.ErrLicenseInactive), errors.Is(err, assignments.ErrDuplicate), errors.Is(err, assignments.ErrNoAvailableSeats):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrLicenseProductMismatch), errors.Is(err, ErrRequesterUnavailable), errors.Is(err, assignments.ErrAssignmentType), errors.Is(err, assignments.ErrTargetUnavailable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
