package main

import (
	"fmt"
	"log"
	"net"
	"os"

	communitypb "github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/genproto/proto"
	communityhandler "github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/handler/grpc"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/repository/postgres"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func NewUserServiceClientForCommSvc(addr string) (userpb.UserServiceClient, *grpc.ClientConn, error) {
    conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil { return nil, nil, fmt.Errorf("comm-svc: failed to dial user service at %s: %w", addr, err) }
    return userpb.NewUserServiceClient(conn), conn, nil
}

func main() {
	err := godotenv.Load()
	if err != nil { log.Println("No .env file found for community-service") }

	repo, err := postgres.NewCommunityRepository()
	if err != nil { log.Fatalf("failed to initialize community repository: %v", err) }

	userSvcAddr := os.Getenv("USER_SERVICE_ADDR"); if userSvcAddr == "" { userSvcAddr = "user-service:50051" }
    userClient, userConn, err := NewUserServiceClientForCommSvc(userSvcAddr)
    if err != nil { log.Fatalf("msg-svc: failed to connect to user-svc: %v", err) }
    defer userConn.Close()

	port := os.Getenv("PORT")
	if port == "" { port = "50057" }
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil { log.Fatalf("failed to listen on port %s: %v", port, err) }

	s := grpc.NewServer()
	communityServer := communityhandler.NewCommunityHandler(repo, userClient)
	communitypb.RegisterCommunityServiceServer(s, communityServer)
	reflection.Register(s)

	fmt.Printf("Community gRPC server listening on :%s\n", port)
	if err := s.Serve(lis); err != nil { log.Fatalf("failed to serve gRPC: %v", err) }
}