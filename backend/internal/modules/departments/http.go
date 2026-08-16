package departments

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
	routes := v1.Group("/departments")
	routes.Use(h.auth.Authenticate())
	routes.GET("", h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager), h.list)
	routes.POST("", h.auth.RequireRoles(auth.RoleAdmin), h.create)
	routes.PUT("/:id", h.auth.RequireRoles(auth.RoleAdmin), h.update)
}

func (h *HTTPHandler) list(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) create(c *gin.Context) {
	input, ok := bindDepartmentInput(c)
	if !ok {
		return
	}
	item, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeDepartmentError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionCreate, audit.EntityDepartment, item.ID, map[string]any{
		"name": item.Name, "code": item.Code,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *HTTPHandler) update(c *gin.Context) {
	input, ok := bindDepartmentInput(c)
	if !ok {
		return
	}
	item, err := h.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeDepartmentError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionUpdate, audit.EntityDepartment, item.ID, map[string]any{
		"name": item.Name, "code": item.Code,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func bindDepartmentInput(c *gin.Context) (Input, bool) {
	var request struct {
		Name string `json:"name" binding:"required"`
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department name and code are required"})
		return Input{}, false
	}
	return Input{Name: request.Name, Code: request.Code}, true
}

func writeDepartmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidData):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNameExists), errors.Is(err, ErrCodeExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
