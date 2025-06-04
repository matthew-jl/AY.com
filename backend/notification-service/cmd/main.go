package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/event"
	notifpb "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/genproto/proto"
	notifhandler "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/handler/grpc"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/repository/postgres"
	notifUtils "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/utils"
	notifWs "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/websocket"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

var jwtSecretBytesNotif []byte

func init() {
    secret := os.Getenv("JWT_SECRET_KEY")
    if secret == "" {
        secret = "your-secret-key"
        log.Println("Warning: JWT_SECRET_KEY not set in notification-service, using insecure default. MUST MATCH API GATEWAY!")
    }
    jwtSecretBytesNotif = []byte(secret)
}

func NewUserServiceClientForNotifSvc(addr string) (userpb.UserServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return nil, nil, fmt.Errorf("notif-svc: failed to dial user service at %s: %w", addr, err)}
	return userpb.NewUserServiceClient(conn), conn, nil
}


func main() {
	err := godotenv.Load(); if err != nil { log.Println("No .env file for notification-service") }

	notifUtils.InitEmailNotifier()

	repo, err := postgres.NewNotificationRepository()
	if err != nil { log.Fatalf("failed to init notif repo: %v", err) }

	userSvcAddr := os.Getenv("USER_SERVICE_ADDR")
	if userSvcAddr == "" { userSvcAddr = "user-service:50051" }
	userSvcClient, userSvcConn, err := NewUserServiceClientForNotifSvc(userSvcAddr)
	if err != nil { log.Fatalf("notif-svc failed to connect to user-svc: %v", err) }
	defer userSvcConn.Close()

	// WebSocket Hub
    wsHub := notifWs.NewHub()
    go wsHub.Run()

	// RabbitMQ Consumer
	consumer, err := event.NewConsumer(repo, userSvcClient, wsHub)
	if err != nil { log.Fatalf("failed to init RabbitMQ consumer: %v", err) }
	defer consumer.Close()
	go consumer.StartConsuming()

	grpcPort := os.Getenv("PORT"); if grpcPort == "" { grpcPort = "50055" }
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil { log.Fatalf("failed to listen (gRPC) on %s: %v", grpcPort, err) }

	s := grpc.NewServer()
	notifServer := notifhandler.NewNotificationHandler(repo)
	notifpb.RegisterNotificationServiceServer(s, notifServer)
	reflection.Register(s)

	fmt.Printf("Notification gRPC server listening on :%s\n", grpcPort)
	go func() {
		if err := s.Serve(lis); err != nil { log.Fatalf("failed to serve gRPC: %v", err) }
	}()

    // WebSocket HTTP Server
    wsPort := os.Getenv("WEBSOCKET_PORT"); if wsPort == "" { wsPort = "8081" }
	mux := http.NewServeMux()
    mux.HandleFunc("/ws/notifications", func(w http.ResponseWriter, r *http.Request) {
        log.Println("WebSocket connection attempt...")
        var tokenString string

        // Option 1: Get token from query parameter (common for WS)
        // e.g., ws://localhost:8081/ws/notifications?token=YOUR_JWT_TOKEN
        tokenQuery := r.URL.Query().Get("token")
        if tokenQuery != "" {
            tokenString = tokenQuery
            log.Println("Token found in query parameter.")
        } else {
            // Option 2: Get token from Authorization header (less common for direct WS, but possible if client can set it)
            authHeader := r.Header.Get("Authorization")
            if strings.HasPrefix(authHeader, "Bearer ") {
                tokenString = strings.TrimPrefix(authHeader, "Bearer ")
                log.Println("Token found in Authorization header.")
            }
        }

        if tokenString == "" {
            log.Println("WebSocket Auth: Token not provided.")
            http.Error(w, "Unauthorized: Token not provided", http.StatusUnauthorized)
            return
        }

        userID, err := notifUtils.ValidateTokenAndGetUserID(tokenString, jwtSecretBytesNotif)
        if err != nil {
            log.Printf("WebSocket Auth: Token validation failed: %v", err)
            http.Error(w, fmt.Sprintf("Unauthorized: %s", err.Error()), http.StatusUnauthorized)
            return
        }

        if userID == 0 { // Should be caught by ValidateTokenAndGetUserID errors
            log.Println("WebSocket Auth: Validated token but UserID is 0.")
            http.Error(w, "Unauthorized: Invalid user in token", http.StatusUnauthorized)
            return
        }

        log.Printf("WebSocket authenticated for UserID: %d", userID)
        notifWs.ServeWs(wsHub, w, r, userID) // Pass the authenticated userID
    })

    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Notification HTTP Server OK"))
    })
    fmt.Printf("Notification WebSocket server listening on :%s\n", wsPort)
    wsServer := &http.Server{
        Addr:    ":" + wsPort,
        Handler: mux,
    }
    if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("failed to start WebSocket server: %v", err)
    }

    select {}
}