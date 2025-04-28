// Placeholder main.go
package main

import (
	"os"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/http"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/handler/websocket"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/route"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize gRPC clients
	userServiceAddr := os.Getenv("USER_SERVICE_ADDR")
	if userServiceAddr == "" {
		userServiceAddr = "user-service:50051"
	}
	userClient, err := client.NewUserClient(userServiceAddr)
	if err != nil {
		logrus.Fatalf("failed to connect to user-service: %v", err)
	}

	// Initialize handlers
	authHandler := http.NewAuthHandler(userClient)
	wsHub := websocket.NewHub()

	// JWT secret (TODO: ganti pake environment variable)
	jwtSecret := "your-secret-key"

	// Set up router
	r := route.SetupRouter(authHandler, wsHub, jwtSecret)

	// Start server
	logrus.Info("API Gateway starting on :8080")
	if err := r.Run(":8080"); err != nil {
		logrus.Fatalf("failed to start server: %v", err)
	}
}