package main

import (
	"fmt"
	"log"
	"net"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	userhandler "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/handler/grpc"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/repository/postgres"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading environment variables directly")
	}

	// Initialize repository
	repo, err := postgres.NewUserRepository()
	if err != nil {
		log.Fatalf("failed to initialize repository: %v", err)
	}

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, userhandler.NewUserHandler(repo))
	reflection.Register(s)

	fmt.Printf("User gRPC server listening on :50051\n")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}