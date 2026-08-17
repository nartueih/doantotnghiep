package httpapi

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.json
var openAPISpec []byte

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Enterprise License Manager API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      if (typeof SwaggerUIBundle === "undefined") {
        document.getElementById("swagger-ui").innerHTML =
          '<h1>Enterprise License Manager API</h1>' +
          '<p>Swagger UI assets could not be loaded. The OpenAPI specification remains available at ' +
          '<a href="/openapi.json">/openapi.json</a>.</p>';
        return;
      }
      SwaggerUIBundle({
        url: "/openapi.json?v=1.0.0",
        dom_id: "#swagger-ui",
        deepLinking: true,
        persistAuthorization: true,
        displayRequestDuration: true
      });
    };
  </script>
</body>
</html>`

func registerDocumentationRoutes(router *gin.Engine) {
	router.GET("/openapi.json", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		c.Data(http.StatusOK, "application/json; charset=utf-8", openAPISpec)
	})
	router.GET("/docs", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
	})
}
