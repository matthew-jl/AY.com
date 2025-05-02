package route

import (
	"net/http"
	"strings"

	gwHTTPHandler "github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/http"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/websocket"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetupRouter(authHandler *gwHTTPHandler.AuthHandler, wsHub *websocket.Hub, jwtSecret string) *gin.Engine {
	r := gin.New()

	// Configure CORS
	config := cors.Config{
		AllowOrigins: []string{"http://localhost:5173"}, // frontend
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,                     
	}

	// Middleware
	r.Use(gin.Recovery()) // Recover from panics
	r.Use(cors.New(config)) // CORS middleware
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

	// Routes
	v1 := r.Group("/api/v1")

	// --- Authentication routes (Should NOT require JWT auth) ---
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/verify", authHandler.VerifyEmail)
	}

	// --- User routes (Requires JWT auth) ---
	users := v1.Group("/users")
	users.Use(middleware.AuthMiddleware(jwtSecret))
	{
		users.GET("/health", authHandler.HealthCheck)
		// users.GET("/profile", authHandler.GetProfile) // Add later
		// users.PUT("/profile", authHandler.UpdateProfile) // Add later
	}

	// --- WebSocket routes (Auth handled potentially within the handler) ---
	ws := v1.Group("/ws")
	ws.Use(middleware.AuthMiddleware(jwtSecret))
	{
		ws.GET("/connect", wsHub.HandleWebSocket)
	}

	// --- Gateway Health Check (Public) ---
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "API Gateway is OK"})
	})

	// Handle NoRoute - 404
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		}
	})

	return r
}