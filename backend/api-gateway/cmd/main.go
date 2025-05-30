// Placeholder main.go
package main

import (
	"os"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	gwHTTPHandler "github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/http"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/websocket"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/route"
	"github.com/sirupsen/logrus"
)

func main() {
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

	logrus.Info("gRPC Clients initialized")

	// Initialize handlers
	authHandler := gwHTTPHandler.NewAuthHandler(userClient)
	threadHandler := gwHTTPHandler.NewThreadHandler(threadClient, mediaClient, userClient)
	mediaHandler := gwHTTPHandler.NewMediaHandler(mediaClient)
	profileHandler := gwHTTPHandler.NewProfileHandler(userClient)
	searchHandler := gwHTTPHandler.NewSearchHandler(searchClient, userClient, threadClient, mediaClient)
	wsHub := websocket.NewHub()

	// JWT secret 
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key" // Default, ensure consistency
		logrus.Warn("JWT_SECRET_KEY not set, using insecure default")
	}

	// Set up router
	r := route.SetupRouter(authHandler, threadHandler, mediaHandler, profileHandler, searchHandler, wsHub, jwtSecret)

	// Start server
	logrus.Info("API Gateway starting on :8080")
	if err := r.Run(":8080"); err != nil {
		logrus.Fatalf("failed to start server: %v", err)
	}
}