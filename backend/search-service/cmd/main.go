package main

import (
	"fmt"
	"log"
	"net"
	"os"

	searchpb "github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/genproto/proto"
	searchhandler "github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/handler/grpc"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/repository"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	err := godotenv.Load()
	if err != nil { log.Println("No .env file found for search-service") }

	repo, err := repository.NewSearchRepository()
	if err != nil { log.Fatalf("failed to initialize search repository: %v", err) }

	port := os.Getenv("PORT")
	if port == "" { port = "50054" }
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil { log.Fatalf("failed to listen on port %s: %v", port, err) }

	s := grpc.NewServer()
	searchServer := searchhandler.NewSearchHandler(repo)
	searchpb.RegisterSearchServiceServer(s, searchServer)
	reflection.Register(s)

	fmt.Printf("Search gRPC server listening on :%s\n", port)
	if err := s.Serve(lis); err != nil { log.Fatalf("failed to serve gRPC: %v", err) }
}