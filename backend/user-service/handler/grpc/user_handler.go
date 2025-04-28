package grpc

import (
	"context"
	"log"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/repository/postgres"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserHandler struct {
	userpb.UnimplementedUserServiceServer
	repo *postgres.UserRepository
}

func NewUserHandler(repo *postgres.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*userpb.HealthResponse, error) {
	log.Printf("Received HealthCheck request")
	if err := h.repo.CheckHealth(ctx); err != nil {
		return &userpb.HealthResponse{Status: "User Service is DEGRADED (DB Error)"}, nil
	}
	return &userpb.HealthResponse{Status: "User Service is OK (DB Connected)"}, nil
}