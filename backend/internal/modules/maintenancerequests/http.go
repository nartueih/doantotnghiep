package maintenancerequests

import (
	"context"
	"errors"
	"net/http"

	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"

	"github.com/gin-gonic/gin"
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
	employee := v1.Group("/me/maintenance-requests")
	employee.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleEmployee))
	employee.GET("", h.listMine)
	employee.POST("", h.create)
	employee.POST("/:id/cancel", h.cancel)

	admin := v1.Group("/maintenance-requests")
	admin.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	admin.GET("", h.listAdmin)
	admin.POST("/:id/accept", h.accept)
	admin.POST("/:id/complete", h.complete)
	admin.POST("/:id/reject", h.reject)
}

func (h *HTTPHandler) listMine(c *gin.Context) {
	result, err := h.service.ListMine(c.Request.Context(), auth.CurrentUserID(c))
	if err != nil {
		writeMaintenanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) create(c *gin.Context) {
	var request struct {
		DeviceID    string `json:"device_id" binding:"required"`
		Category    string `json:"category" binding:"required"`
		Priority    string `json:"priority" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeCodedMaintenanceError(c, http.StatusBadRequest, "invalid_request", ErrInvalidData)
		return
	}
	h.runMutation(c, http.StatusCreated, audit.ActionRequest,
		func(ctx context.Context) (Request, error) {
			return h.service.Create(ctx, auth.CurrentUserID(c), CreateInput{
				DeviceID: request.DeviceID, Category: request.Category, Priority: request.Priority,
				Title: request.Title, Description: request.Description,
			})
		},
		func(item Request) map[string]any {
			return map[string]any{"device_id": item.DeviceID, "category": item.Category, "priority": item.Priority, "status": item.Status}
		},
	)
}

func (h *HTTPHandler) cancel(c *gin.Context) {
	h.runMutation(c, http.StatusOK, audit.ActionCancel,
		func(ctx context.Context) (Request, error) {
			return h.service.Cancel(ctx, auth.CurrentUserID(c), c.Param("id"))
		},
		func(item Request) map[string]any {
			return map[string]any{"device_id": item.DeviceID, "status": item.Status}
		},
	)
}

func (h *HTTPHandler) listAdmin(c *gin.Context) {
	items, err := h.service.ListAdmin(c.Request.Context(), Filter{
		Status: c.Query("status"), Priority: c.Query("priority"), Category: c.Query("category"), Search: c.Query("search"),
	})
	if err != nil {
		writeMaintenanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) accept(c *gin.Context) {
	h.runMutation(c, http.StatusOK, audit.ActionAccept,
		func(ctx context.Context) (Request, error) {
			return h.service.Accept(ctx, auth.CurrentUserID(c), c.Param("id"))
		},
		func(item Request) map[string]any {
			return map[string]any{"device_id": item.DeviceID, "assigned_to": item.AssignedTo, "status": item.Status}
		},
	)
}

func (h *HTTPHandler) complete(c *gin.Context) {
	var request struct {
		ResponseNote string `json:"response_note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeCodedMaintenanceError(c, http.StatusBadRequest, "invalid_request", ErrInvalidData)
		return
	}
	h.runMutation(c, http.StatusOK, audit.ActionComplete,
		func(ctx context.Context) (Request, error) {
			return h.service.Complete(ctx, auth.CurrentUserID(c), c.Param("id"), CompleteInput{ResponseNote: request.ResponseNote})
		},
		func(item Request) map[string]any {
			return map[string]any{"device_id": item.DeviceID, "status": item.Status}
		},
	)
}

func (h *HTTPHandler) reject(c *gin.Context) {
	var request struct {
		ResponseNote string `json:"response_note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeCodedMaintenanceError(c, http.StatusBadRequest, "invalid_request", ErrInvalidData)
		return
	}
	h.runMutation(c, http.StatusOK, audit.ActionReject,
		func(ctx context.Context) (Request, error) {
			return h.service.Reject(ctx, auth.CurrentUserID(c), c.Param("id"), RejectInput{ResponseNote: request.ResponseNote})
		},
		func(item Request) map[string]any {
			return map[string]any{"device_id": item.DeviceID, "status": item.Status}
		},
	)
}

func (h *HTTPHandler) runMutation(c *gin.Context, responseStatus int, action string, operation func(context.Context) (Request, error), metadata func(Request) map[string]any) {
	var item Request
	err := h.service.transactions.WithinTransaction(c.Request.Context(), func(txCtx context.Context) error {
		var err error
		item, err = operation(txCtx)
		if err != nil || h.audit == nil {
			return err
		}
		_, err = h.audit.Record(txCtx, audit.RecordInput{
			ActorID: auth.CurrentUserID(c), Action: action, EntityType: audit.EntityMaintenanceRequest,
			EntityID: item.ID, Metadata: metadata(item), IPAddress: c.ClientIP(),
		})
		return err
	})
	if err != nil {
		writeMaintenanceError(c, err)
		return
	}
	c.JSON(responseStatus, item)
}

func writeMaintenanceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidData), errors.Is(err, ErrInvalidCategory), errors.Is(err, ErrInvalidPriority):
		writeCodedMaintenanceError(c, http.StatusBadRequest, "invalid_request", err)
	case errors.Is(err, ErrReviewerUnavailable):
		writeCodedMaintenanceError(c, http.StatusForbidden, "reviewer_unavailable", err)
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrDeviceNotFound):
		writeCodedMaintenanceError(c, http.StatusNotFound, "not_found", err)
	case errors.Is(err, ErrOpenDuplicate):
		writeCodedMaintenanceError(c, http.StatusConflict, "open_maintenance_request_exists", err)
	case errors.Is(err, ErrInvalidState):
		writeCodedMaintenanceError(c, http.StatusConflict, "invalid_maintenance_state", err)
	case errors.Is(err, ErrRequesterUnavailable):
		writeCodedMaintenanceError(c, http.StatusUnprocessableEntity, "requester_unavailable", err)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func writeCodedMaintenanceError(c *gin.Context, status int, code string, err error) {
	c.JSON(status, gin.H{"error": err.Error(), "code": code})
}
