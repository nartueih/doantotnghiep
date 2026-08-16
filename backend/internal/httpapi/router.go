package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
)

type PingFunc func(context.Context) error

func NewRouter(
	ping PingFunc,
	authHandler *auth.HTTPHandler,
	usersHandler *users.HTTPHandler,
	softwareHandler *software.HTTPHandler,
	licenseHandler *licenses.HTTPHandler,
) http.Handler {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

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
	if usersHandler != nil {
		usersHandler.RegisterRoutes(v1)
	}
	if softwareHandler != nil {
		softwareHandler.RegisterRoutes(v1)
	}
	if licenseHandler != nil {
		licenseHandler.RegisterRoutes(v1)
	}

	return router
}
