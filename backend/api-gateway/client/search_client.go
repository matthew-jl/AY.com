package client

import (
	"context"
	"fmt"

	searchpb "github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type SearchClient struct {
	client searchpb.SearchServiceClient
	conn   *grpc.ClientConn
}

func NewSearchClient(addr string) (*SearchClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial search service at %s: %w", addr, err)
	}
	client := searchpb.NewSearchServiceClient(conn)
	return &SearchClient{client: client, conn: conn}, nil
}

func (c *SearchClient) Close() error {
	if c.conn != nil { return c.conn.Close() }
	return nil
}

func (c *SearchClient) HealthCheck(ctx context.Context) (*searchpb.HealthResponse, error) {
	return c.client.HealthCheck(ctx, &emptypb.Empty{})
}

func (c *SearchClient) SearchUsers(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchUserIDsResponse, error) {
	return c.client.SearchUsers(ctx, req)
}

func (c *SearchClient) SearchThreads(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchThreadIDsResponse, error) {
	return c.client.SearchThreads(ctx, req)
}

func (c *SearchClient) GetTrendingHashtags(ctx context.Context, req *searchpb.GetTrendingHashtagsRequest) (*searchpb.GetTrendingHashtagsResponse, error) {
	return c.client.GetTrendingHashtags(ctx, req)
}

func (c *SearchClient) GetTopUsersToFollow(ctx context.Context, req *searchpb.GetTopUsersToFollowRequest) (*searchpb.SearchUserIDsResponse, error) {
	return c.client.GetTopUsersToFollow(ctx, req)
}