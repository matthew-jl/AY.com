package client

import (
	"context"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserClient struct {
	client userpb.UserServiceClient
	conn   *grpc.ClientConn
}

func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := userpb.NewUserServiceClient(conn)
	return &UserClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *UserClient) HealthCheck(ctx context.Context) (*userpb.HealthResponse, error) {
	return c.client.HealthCheck(ctx, &emptypb.Empty{})
}

func (c *UserClient) Register(ctx context.Context, req *userpb.RegisterRequest) (*emptypb.Empty, error) {
	return c.client.Register(ctx, req)
}

func (c *UserClient) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.AuthResponse, error) {
	return c.client.Login(ctx, req)
}

func (c *UserClient) VerifyEmail(ctx context.Context, req *userpb.VerifyEmailRequest) (*emptypb.Empty, error) {
	return c.client.VerifyEmail(ctx, req)
}

func (c *UserClient) GetSecurityQuestion(ctx context.Context, req *userpb.GetSecurityQuestionRequest) (*userpb.GetSecurityQuestionResponse, error) {
	return c.client.GetSecurityQuestion(ctx, req)
}

func (c *UserClient) ResetPassword(ctx context.Context, req *userpb.ResetPasswordRequest) (*emptypb.Empty, error) {
	return c.client.ResetPassword(ctx, req)
}

func (c *UserClient) GetUserProfile(ctx context.Context, req *userpb.GetUserProfileRequest) (*userpb.User, error) {
	return c.client.GetUserProfile(ctx, req)
}