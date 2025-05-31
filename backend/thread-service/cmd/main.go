package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/client"
	threadpb "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/genproto/proto"
	threadhandler "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/handler/grpc"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/repository/postgres"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	err := godotenv.Load()
	if err != nil { log.Println("No .env file found for thread-service") }

	repo, err := postgres.NewThreadRepository()
	if err != nil { log.Fatalf("failed to initialize thread repository: %v", err) }

	// Initialize user client
	userServiceAddr := os.Getenv("USER_SERVICE_ADDR")
	if userServiceAddr == "" {
		userServiceAddr = "user-service:50051" // Default to user-service container name and port
	}
	userClient, err := client.NewUserClient(userServiceAddr)
	if err != nil {
		log.Fatalf("failed to initialize user client: %v", err)
	}
	defer userClient.Close()

	// Initialize search client
	searchServiceAddr := os.Getenv("SEARCH_SERVICE_ADDR")
	if searchServiceAddr == "" {
		searchServiceAddr = "search-service:50054" // Default to search-service container name and port
	}
	searchClient, err := client.NewSearchClient(searchServiceAddr)
	if err != nil {
		log.Fatalf("failed to initialize search client: %v", err)
	}
	defer searchClient.Close()

	port := os.Getenv("PORT")
	if port == "" { port = "50052" } // Default thread service port
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil { log.Fatalf("failed to listen on port %s: %v", port, err) }

	s := grpc.NewServer()
	threadServer := threadhandler.NewThreadHandler(repo, userClient.GetClient(), searchClient.GetClient())
	threadpb.RegisterThreadServiceServer(s, threadServer)
	reflection.Register(s)

	fmt.Printf("Thread gRPC server listening on :%s\n", port)
	if err := s.Serve(lis); err != nil { log.Fatalf("failed to serve gRPC: %v", err) }
}