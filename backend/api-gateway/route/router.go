package route

import (
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/http"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/websocket"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetupRouter(authHandler *http.AuthHandler, wsHub *websocket.Hub, jwtSecret string) *gin.Engine {
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery()) // Recover from panics
	r.Use(func(c *gin.Context) {
		// Custom logging with logrus
		logrus.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		}).Info("HTTP Request")
		c.Next()
		if len(c.Errors) > 0 {
			logrus.Error(c.Errors.String())
		}
	})
	r.Use(middleware.AuthMiddleware(jwtSecret))

	// Routes
	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.GET("/health", authHandler.HealthCheck)
		}
		ws := v1.Group("/ws")
		{
			ws.GET("/connect", wsHub.HandleWebSocket)
		}
	}

	return r
}