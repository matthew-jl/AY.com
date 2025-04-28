package client

import (
	"context"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserClient struct {
	client userpb.UserServiceClient
}

func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &UserClient{client: userpb.NewUserServiceClient(conn)}, nil
}

func (c *UserClient) HealthCheck(ctx context.Context) (*userpb.HealthResponse, error) {
	return c.client.HealthCheck(ctx, &emptypb.Empty{})
}