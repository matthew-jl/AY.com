package main

import (
	"context"
	"fmt"
	"log"
	"net"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/genproto/user"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	userpb.UnimplementedUserServiceServer
}

func (s *server) HealthCheck(ctx context.Context, in *emptypb.Empty) (*userpb.HealthResponse, error) {
	log.Printf("Received HealthCheck request")
	return &userpb.HealthResponse{Status: "User Service is OK"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, &server{})

	fmt.Printf("User gRPC server listening on :50051\n")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}