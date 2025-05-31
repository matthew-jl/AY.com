package client

import (
	"context"
	"fmt"

	searchpb "github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SearchClient struct {
	client searchpb.SearchServiceClient
	conn   *grpc.ClientConn
}

func NewSearchClient(addr string) (*SearchClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial search service at %s: %w", addr, err)
	}
	client := searchpb.NewSearchServiceClient(conn)
	return &SearchClient{client: client, conn: conn}, nil
}

func (c *SearchClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (sc *SearchClient) GetClient() searchpb.SearchServiceClient {
	return sc.client
}

func (c *SearchClient) IncrementHashtagCounts(ctx context.Context, hashtags []string) error {
	if len(hashtags) == 0 {
		return nil
	}
	req := &searchpb.IncrementHashtagCountsRequest{
		Hashtags: hashtags,
	}
	_, err := c.client.IncrementHashtagCounts(ctx, req)
	return err
}