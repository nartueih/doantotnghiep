package software

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
	routes := v1.Group("/software")
	routes.Use(h.auth.Authenticate(), h.auth.RequireRoles(auth.RoleAdmin, auth.RoleITManager))
	routes.GET("", h.list)
	routes.POST("", h.create)
	routes.PUT("/:id", h.update)
}

func (h *HTTPHandler) list(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HTTPHandler) create(c *gin.Context) {
	input, ok := bindInput(c)
	if !ok {
		return
	}
	product, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionCreate, audit.EntitySoftware, product.ID, map[string]any{
		"name": product.Name, "publisher": product.Publisher, "version": product.Version,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusCreated, product)
}

func (h *HTTPHandler) update(c *gin.Context) {
	input, ok := bindInput(c)
	if !ok {
		return
	}
	product, err := h.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	if err := audit.RecordRequest(c, h.audit, audit.ActionUpdate, audit.EntitySoftware, product.ID, map[string]any{
		"name": product.Name, "publisher": product.Publisher, "version": product.Version,
	}); err != nil {
		audit.WriteError(c)
		return
	}
	c.JSON(http.StatusOK, product)
}

func bindInput(c *gin.Context) (Input, bool) {
	var request struct {
		Name        string `json:"name" binding:"required"`
		Publisher   string `json:"publisher" binding:"required"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and publisher are required"})
		return Input{}, false
	}
	return Input{
		Name:        request.Name,
		Publisher:   request.Publisher,
		Version:     request.Version,
		Description: request.Description,
	}, true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidData):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
