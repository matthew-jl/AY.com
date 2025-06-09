package main

import (
	"os"
	"strings"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	gwHTTPHandler "github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/http"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/websocket"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/route"
	"github.com/sirupsen/logrus"

	_ "github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/docs"
)

func initLogger() {
	logLevel := logrus.InfoLevel

	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	if env == "" {
		env = "development"
	}

	switch env {
	case "production":
		logLevel = logrus.WarnLevel // warnings and errors
	case "development":
		logLevel = logrus.DebugLevel // debug, info, warnings, and errors
	default:
		logrus.Warnf("Unknown environment '%s', defaulting to Info level", env)
	}

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	// Initialize and test logger (logrus)
	initLogger()
	logrus.Debug("This is a debug message")
	logrus.Info("This is an info message")
	logrus.Warn("This is a warning message")
	logrus.Error("This is an error message")

	// Initialize gRPC clients
	userServiceAddr := os.Getenv("USER_SERVICE_ADDR")
	if userServiceAddr == "" { userServiceAddr = "user-service:50051" }
	userClient, err := client.NewUserClient(userServiceAddr)
	if err != nil { logrus.Fatalf("failed to connect to user-service: %v", err) }
	defer userClient.Close()

	threadServiceAddr := os.Getenv("THREAD_SERVICE_ADDR")
	if threadServiceAddr == "" { threadServiceAddr = "thread-service:50052" }
	threadClient, err := client.NewThreadClient(threadServiceAddr)
	if err != nil { logrus.Fatalf("failed to connect to thread-service: %v", err) }
	defer threadClient.Close()

	mediaServiceAddr := os.Getenv("MEDIA_SERVICE_ADDR")
	if mediaServiceAddr == "" { mediaServiceAddr = "media-service:50053" }
	mediaClient, err := client.NewMediaClient(mediaServiceAddr)
	if err != nil { logrus.Fatalf("failed to connect to media-service: %v", err) }
	defer mediaClient.Close()

	searchServiceAddr := os.Getenv("SEARCH_SERVICE_ADDR")
	if searchServiceAddr == "" { searchServiceAddr = "search-service:50054" }
	searchClient, err := client.NewSearchClient(searchServiceAddr)
	if err != nil { logrus.Fatalf("failed to connect to search-service: %v", err) }
	defer searchClient.Close()

	notificationServiceAddr := os.Getenv("NOTIFICATION_SERVICE_ADDR")
	if notificationServiceAddr == "" { notificationServiceAddr = "notification-service:50055" }
	notificationClient, err := client.NewNotificationClient(notificationServiceAddr)
	if err != nil { logrus.Fatalf("failed to connect to notification-service: %v", err) }
	defer notificationClient.Close()

	messageServiceAddr := os.Getenv("MESSAGE_SERVICE_ADDR")
	if messageServiceAddr == "" { messageServiceAddr = "message-service:50056" }
	messageClient, err := client.NewMessageClient(messageServiceAddr)
	if err != nil { logrus.Fatalf("failed to connect to message-service: %v", err) }
	defer messageClient.Close()

	communityServiceAddr := os.Getenv("COMMUNITY_SERVICE_ADDR")
	if communityServiceAddr == "" { communityServiceAddr = "community-service:50057" }
	communityClient, err := client.NewCommunityClient(communityServiceAddr)
	if err != nil { logrus.Fatalf("failed to connect to community-service: %v", err) }
	defer communityClient.Close()

	logrus.Info("gRPC Clients initialized")

	// Initialize handlers
	authHandler := gwHTTPHandler.NewAuthHandler(userClient)
	threadHandler := gwHTTPHandler.NewThreadHandler(threadClient, mediaClient, userClient)
	mediaHandler := gwHTTPHandler.NewMediaHandler(mediaClient)
	profileHandler := gwHTTPHandler.NewProfileHandler(userClient)
	searchHandler := gwHTTPHandler.NewSearchHandler(searchClient, userClient, threadClient, mediaClient)
	notificationHandler := gwHTTPHandler.NewNotificationHandler(notificationClient)
	messageHandler := gwHTTPHandler.NewMessageHandler(messageClient, userClient, mediaClient)
	communityHandler := gwHTTPHandler.NewCommunityHandler(communityClient, userClient)
	wsHub := websocket.NewHub()

	logrus.Info("HTTP Handlers initialized")

	// JWT secret 
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key"
		logrus.Warn("JWT_SECRET_KEY not set, using insecure default")
	}

	// Set up router
	r := route.SetupRouter(authHandler, threadHandler, mediaHandler, profileHandler, searchHandler, notificationHandler, messageHandler, communityHandler, wsHub, jwtSecret)

	// Start server
	logrus.Info("API Gateway starting on :8080")
	if err := r.Run(":8080"); err != nil {
		logrus.Fatalf("failed to start server: %v", err)
	}
}