package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/genproto/user"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type server struct {
	userpb.UnimplementedUserServiceServer
	db *gorm.DB
}

func (s *server) HealthCheck(ctx context.Context, in *emptypb.Empty) (*userpb.HealthResponse, error) {
	log.Printf("Received HealthCheck request")
	var count int64
	if err := s.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return &userpb.HealthResponse{Status: "User Service is DEGRADED (DB Error)"}, nil
	}
	return &userpb.HealthResponse{Status: "User Service is OK (DB Connected)"}, nil
}

func main() {
	// Initialize database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Auto-migrate the User model
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("failed to auto-migrate: %v", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	server := &server{db: db}
	userpb.RegisterUserServiceServer(s, server)
	reflection.Register(s)

	fmt.Printf("User gRPC server listening on :50051\n")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}