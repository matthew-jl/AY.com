package client

import (
	"context"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func (uc *UserClient) GetClient() userpb.UserServiceClient {
	return uc.client
}

func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *UserClient) GetUserByUsername(ctx context.Context, req *userpb.GetUserByUsernameRequest) (*userpb.User, error) {
	return c.client.GetUserByUsername(ctx, req)
}