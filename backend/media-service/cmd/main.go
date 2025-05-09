package main

import (
	"fmt"
	"log"
	"net"
	"os"

	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	mediahandler "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/handler/grpc"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/repository/postgres"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	err := godotenv.Load() // Load .env file for local dev
	if err != nil { log.Println("No .env file found for media-service") }

	repo, err := postgres.NewMediaRepository()
	if err != nil { log.Fatalf("failed to initialize media repository: %v", err) }

	port := os.Getenv("PORT")
	if port == "" { port = "50053" } // Default media service port
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil { log.Fatalf("failed to listen on port %s: %v", port, err) }

	s := grpc.NewServer()
	mediaServer := mediahandler.NewMediaHandler(repo)
	mediapb.RegisterMediaServiceServer(s, mediaServer)
	reflection.Register(s)

	fmt.Printf("Media gRPC server listening on :%s\n", port)
	if err := s.Serve(lis); err != nil { log.Fatalf("failed to serve gRPC: %v", err) }
}