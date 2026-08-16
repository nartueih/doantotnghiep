package users

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
	routes := v1.Group("/users")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin))
	routes.GET("", h.list)
	routes.POST("", h.create)
	routes.PATCH("/:id/status", h.updateStatus)
}

func (h *HTTPHandler) list(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) create(c *gin.Context) {
	var request struct {
		Email        string `json:"email" binding:"required,email"`
		Password     string `json:"password" binding:"required"`
		FullName     string `json:"full_name" binding:"required"`
		EmployeeCode string `json:"employee_code" binding:"required"`
		DepartmentID string `json:"department_id"`
		Role         string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, password, full_name, employee_code and role are required"})
		return
	}

	user, err := h.service.Create(c.Request.Context(), CreateInput{
		Email:        request.Email,
		Password:     request.Password,
		FullName:     request.FullName,
		EmployeeCode: request.EmployeeCode,
		DepartmentID: request.DepartmentID,
		Role:         request.Role,
	})
	if err != nil {
		writeUserError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionCreate, audit.EntityUser, user.ID, map[string]any{
		"email": user.Email, "role": user.Role, "department_id": user.DepartmentID,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *HTTPHandler) updateStatus(c *gin.Context) {
	var request struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	user, err := h.service.UpdateStatus(
		c.Request.Context(),
		auth.CurrentUserID(c),
		c.Param("id"),
		request.Status,
	)
	if err != nil {
		writeUserError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionStatusChange, audit.EntityUser, user.ID, map[string]any{
		"status": user.Status,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, user)
}

func writeUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrIncompleteUserData), errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrWeakPassword):
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must contain at least 10 characters, including uppercase, lowercase and a number"})
	case errors.Is(err, ErrCannotLockSelf):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrDepartmentNotFound):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrEmailAlreadyExists), errors.Is(err, auth.ErrCodeAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
