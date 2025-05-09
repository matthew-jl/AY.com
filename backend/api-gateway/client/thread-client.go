package client

import (
	"context"
	"fmt"

	threadpb "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type ThreadClient struct {
	client threadpb.ThreadServiceClient
	conn   *grpc.ClientConn
}

func NewThreadClient(addr string) (*ThreadClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial thread service at %s: %w", addr, err)
	}
	client := threadpb.NewThreadServiceClient(conn)
	return &ThreadClient{client: client, conn: conn}, nil
}

func (c *ThreadClient) Close() error {
	if c.conn != nil { return c.conn.Close() }
	return nil
}

func (c *ThreadClient) CreateThread(ctx context.Context, req *threadpb.CreateThreadRequest) (*threadpb.Thread, error) {
	return c.client.CreateThread(ctx, req)
}

func (c *ThreadClient) GetThread(ctx context.Context, req *threadpb.GetThreadRequest) (*threadpb.Thread, error) {
	return c.client.GetThread(ctx, req)
}

func (c *ThreadClient) DeleteThread(ctx context.Context, req *threadpb.DeleteThreadRequest) (*emptypb.Empty, error) {
	return c.client.DeleteThread(ctx, req)
}

func (c *ThreadClient) LikeThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
	return c.client.LikeThread(ctx, req)
}

func (c *ThreadClient) UnlikeThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
	return c.client.UnlikeThread(ctx, req)
}

func (c *ThreadClient) BookmarkThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
	return c.client.BookmarkThread(ctx, req)
}

func (c *ThreadClient) UnbookmarkThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
	return c.client.UnbookmarkThread(ctx, req)
}

func (c *ThreadClient) GetFeedThreads(ctx context.Context, req *threadpb.GetFeedThreadsRequest) (*threadpb.GetFeedThreadsResponse, error) {
	return c.client.GetFeedThreads(ctx, req)
}

// Add other methods like GetUserThreads, GetFeed etc. later