package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/dashboard"
	"license-manager/backend/internal/modules/departments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenserequests"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/maintenancerequests"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/modules/selfservice"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
)

type PingFunc func(context.Context) error

func NewRouter(
	ping PingFunc,
	authHandler *auth.HTTPHandler,
	auditHandler *audit.HTTPHandler,
	dashboardHandler *dashboard.HTTPHandler,
	selfServiceHandler *selfservice.HTTPHandler,
	notificationHandler *notifications.HTTPHandler,
	licenseRequestHandler *licenserequests.HTTPHandler,
	maintenanceRequestHandler *maintenancerequests.HTTPHandler,
	usersHandler *users.HTTPHandler,
	departmentHandler *departments.HTTPHandler,
	softwareHandler *software.HTTPHandler,
	licenseHandler *licenses.HTTPHandler,
	deviceHandler *devices.HTTPHandler,
	assignmentHandler *assignments.HTTPHandler,
) http.Handler {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	registerDocumentationRoutes(router)

	health := router.Group("/health")
	health.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	health.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := router.Group("/api/v1")
	v1.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "enterprise-license-manager",
			"version": "v1",
		})
	})
	if authHandler != nil {
		authHandler.RegisterRoutes(v1)
	}
	if auditHandler != nil {
		auditHandler.RegisterRoutes(v1)
	}
	if dashboardHandler != nil {
		dashboardHandler.RegisterRoutes(v1)
	}
	if selfServiceHandler != nil {
		selfServiceHandler.RegisterRoutes(v1)
	}
	if notificationHandler != nil {
		notificationHandler.RegisterRoutes(v1)
	}
	if licenseRequestHandler != nil {
		licenseRequestHandler.RegisterRoutes(v1)
	}
	if maintenanceRequestHandler != nil {
		maintenanceRequestHandler.RegisterRoutes(v1)
	}
	if usersHandler != nil {
		usersHandler.RegisterRoutes(v1)
	}
	if departmentHandler != nil {
		departmentHandler.RegisterRoutes(v1)
	}
	if softwareHandler != nil {
		softwareHandler.RegisterRoutes(v1)
	}
	if licenseHandler != nil {
		licenseHandler.RegisterRoutes(v1)
	}
	if deviceHandler != nil {
		deviceHandler.RegisterRoutes(v1)
	}
	if assignmentHandler != nil {
		assignmentHandler.RegisterRoutes(v1)
	}

	return router
}
