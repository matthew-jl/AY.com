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

func SetupRouter(
	authHandler *gwHTTPHandler.AuthHandler, 
	threadHandler *gwHTTPHandler.ThreadHandler, 
	mediaHandler *gwHTTPHandler.MediaHandler, 
	wsHub *websocket.Hub, 
	jwtSecret string) *gin.Engine {
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

	authMiddleware := middleware.AuthMiddleware(jwtSecret)

	// Routes
	v1 := r.Group("/api/v1")

	// --- Authentication routes (Should NOT require JWT auth) ---
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/verify", authHandler.VerifyEmail)
		auth.POST("/verify/resend", authHandler.ResendVerificationCode)
		auth.POST("/forgot-password/question", authHandler.GetSecurityQuestion)
		auth.POST("/forgot-password/reset", authHandler.ResetPassword) 
	}

	// --- User routes (Requires JWT auth) ---
	users := v1.Group("/users")
	users.Use(authMiddleware)
	{
		users.GET("/health", authHandler.HealthCheck)
		users.GET("/profile", authHandler.GetProfile)
		// users.PUT("/profile", authHandler.UpdateProfile) // Add later
	}

	// --- Protected Thread Routes ---
	threads := v1.Group("/threads")
	threads.Use(authMiddleware) // Apply auth here
	{
		threads.POST("", threadHandler.CreateThread)
		threads.GET("/feed", threadHandler.GetFeed)
		threads.GET("/:threadId", threadHandler.GetThread)
		threads.DELETE("/:threadId", threadHandler.DeleteThread)

		// Interactions
		threads.POST("/:threadId/like", threadHandler.LikeThread)
		threads.DELETE("/:threadId/like", threadHandler.UnlikeThread)
		threads.POST("/:threadId/bookmark", threadHandler.BookmarkThread)
		threads.DELETE("/:threadId/bookmark", threadHandler.UnbookmarkThread)
		// Add repost routes later
	}

	// --- Public Media Routes ---
	media := v1.Group("/media")
	{
		media.POST("/upload", mediaHandler.UploadMedia) // Upload media file
	}
        // Maybe GET /media/:mediaId/metadata later if needed by frontend directly

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