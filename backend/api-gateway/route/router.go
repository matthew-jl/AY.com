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
	profileHandler *gwHTTPHandler.ProfileHandler,
	searchHandler *gwHTTPHandler.SearchHandler,
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
	r.Use(gin.Recovery())
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
	attemptAuthMiddleware := middleware.AttemptAuthMiddleware(jwtSecret)

	// Routes
	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/verify", authHandler.VerifyEmail)
		auth.POST("/verify/resend", authHandler.ResendVerificationCode)
		auth.POST("/forgot-password/question", authHandler.GetSecurityQuestion)
		auth.POST("/forgot-password/reset", authHandler.ResetPassword) 
	}

	users := v1.Group("/users")
	users.Use(authMiddleware)
	{
		users.GET("/health", authHandler.HealthCheck)
		users.GET("/me/profile", authHandler.GetProfile)
		users.PUT("/me/profile", authHandler.UpdateOwnUserProfile)
	}

	userProfiles := v1.Group("/profiles")
	userProfiles.Use(attemptAuthMiddleware)
	{
		userProfiles.GET("/:username", profileHandler.GetUserProfileByUsername)
		userProfiles.GET("/:username/followers", profileHandler.GetFollowers)
		userProfiles.GET("/:username/following", profileHandler.GetFollowing)
		userProfiles.GET("/:username/threads", threadHandler.GetUserSpecificThreads)

		// Actions requiring auth
		userProfiles.POST("/:username/follow", authMiddleware, profileHandler.FollowUser)
		userProfiles.DELETE("/:username/follow", authMiddleware, profileHandler.UnfollowUser)
		userProfiles.POST("/:username/block", authMiddleware, profileHandler.BlockUser)
		userProfiles.DELETE("/:username/block", authMiddleware, profileHandler.UnblockUser)
	}

	threads := v1.Group("/threads")
	threads.Use(authMiddleware)
	{
		threads.POST("", threadHandler.CreateThread)
		threads.GET("/feed", threadHandler.GetFeed)
		threads.GET("/bookmarked", threadHandler.GetBookmarkedThreadsHTTP)
		threads.GET("/:threadId", threadHandler.GetThread)
		threads.DELETE("/:threadId", threadHandler.DeleteThread)

		threads.POST("/:threadId/like", threadHandler.LikeThread)
		threads.DELETE("/:threadId/like", threadHandler.UnlikeThread)
		threads.POST("/:threadId/bookmark", threadHandler.BookmarkThread)
		threads.DELETE("/:threadId/bookmark", threadHandler.UnbookmarkThread)
		// Add repost routes later
	}

	media := v1.Group("/media")
	media.Use(attemptAuthMiddleware)
	{
		media.POST("/upload", mediaHandler.UploadMedia)
	}
        // Maybe GET /media/:mediaId/metadata later if needed by frontend directly

		search := v1.Group("/search")
		search.Use(attemptAuthMiddleware)
		{
			search.GET("/users", searchHandler.SearchUsersHTTP)
			search.GET("/threads", searchHandler.SearchThreadsHTTP)
			// search.GET("/communities", searchHandler.SearchCommunitiesHTTP) // Add later
		}

		trending := v1.Group("/trending")
		trending.Use(attemptAuthMiddleware)
		{
			trending.GET("/hashtags", searchHandler.GetTrendingHashtagsHTTP)
		}

	// --- WebSocket routes (Auth handled potentially within the handler) ---
	ws := v1.Group("/ws")
	ws.Use(middleware.AuthMiddleware(jwtSecret))
	{
		ws.GET("/connect", wsHub.HandleWebSocket)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "API Gateway is OK"})
	})

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		}
	})

	return r
}