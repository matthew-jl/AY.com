package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	// "github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/event"
	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	messagepb "github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/genproto/proto"
	msghandler "github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/handler/grpc"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/repository/postgres"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/utils"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/websocket"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

var jwtSecretBytesMsg []byte

func init() {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" { secret = "your-very-insecure-default-secret-key"; log.Println("MSG-SVC: JWT_SECRET_KEY not set, using default.") }
	jwtSecretBytesMsg = []byte(secret)
}

func NewUserServiceClientForMsgSvc(addr string) (userpb.UserServiceClient, *grpc.ClientConn, error) {
    conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil { return nil, nil, fmt.Errorf("msg-svc: failed to dial user service at %s: %w", addr, err) }
    return userpb.NewUserServiceClient(conn), conn, nil
}

func NewMediaServiceClientForMsgSvc(addr string) (mediapb.MediaServiceClient, *grpc.ClientConn, error) {
    conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil { return nil, nil, fmt.Errorf("msg-svc: failed to dial media service at %s: %w", addr, err)}
    return mediapb.NewMediaServiceClient(conn), conn, nil
}

func main() {
	err := godotenv.Load(); if err != nil { log.Println("No .env file for message-service") }

	repo, err := postgres.NewMessageRepository()
	if err != nil { log.Fatalf("failed to init message repo: %v", err) }

    userSvcAddr := os.Getenv("USER_SERVICE_ADDR"); if userSvcAddr == "" { userSvcAddr = "user-service:50051" }
    userClient, userConn, err := NewUserServiceClientForMsgSvc(userSvcAddr)
    if err != nil { log.Fatalf("msg-svc: failed to connect to user-svc: %v", err) }
    defer userConn.Close()

	mediaSvcAddr := os.Getenv("MEDIA_SERVICE_ADDR"); if mediaSvcAddr == "" { mediaSvcAddr = "media-service:50053"}
    mediaClient, mediaConn, err := NewMediaServiceClientForMsgSvc(mediaSvcAddr)
    if err != nil { log.Fatalf("msg-svc: failed to connect to media-svc: %v", err)}
    defer mediaConn.Close()


	wsHub := websocket.NewHub() // Create instance of your WebSocket Hub
	go wsHub.Run()              // Run the hub in a goroutine

	// Pass repo, wsHub, and userClient to NewMessageHandler
	msgServer := msghandler.NewMessageHandler(repo, wsHub, userClient, mediaClient)

	// gRPC Server
	grpcPort := os.Getenv("PORT"); if grpcPort == "" { grpcPort = "50056" }
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil { log.Fatalf("failed to listen (gRPC) on %s: %v", grpcPort, err) }
	s := grpc.NewServer()
	messagepb.RegisterMessageServiceServer(s, msgServer)
	reflection.Register(s)
	fmt.Printf("Message gRPC server listening on :%s\n", grpcPort)
	go func() { if err := s.Serve(lis); err != nil { log.Fatalf("failed to serve gRPC: %v", err) } }()

	// WebSocket HTTP Server
	wsPort := os.Getenv("WEBSOCKET_PORT"); if wsPort == "" { wsPort = "8082" }
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/ws/messages", func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.URL.Query().Get("token")
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") { tokenString = strings.TrimPrefix(authHeader, "Bearer ") }
		}
		if tokenString == "" { http.Error(w, "Unauthorized: Token missing", http.StatusUnauthorized); return }

		userID, err := utils.ValidateTokenAndGetUserID(tokenString, jwtSecretBytesMsg)
		if err != nil { http.Error(w, fmt.Sprintf("Unauthorized: %s", err.Error()), http.StatusUnauthorized); return }
		if userID == 0 { http.Error(w, "Unauthorized: Invalid user in token", http.StatusUnauthorized); return }

		log.Printf("WebSocket connection for messaging authenticated for UserID: %d", userID)
		websocket.ServeWs(wsHub, w, r, userID)
	})
    httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(http.StatusOK); w.Write([]byte("Message HTTP OK"))})

	fmt.Printf("Message WebSocket server listening on :%s\n", wsPort)
	if err := http.ListenAndServe(":"+wsPort, httpMux); err != nil { log.Fatalf("failed to start Message WebSocket server: %v", err) }

	select {}
}