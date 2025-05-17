package client

import (
	"context"
	"fmt"

	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MediaClient struct {
	client mediapb.MediaServiceClient
	conn   *grpc.ClientConn
}

func NewMediaClient(addr string) (*MediaClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial media service at %s: %w", addr, err)
	}
	client := mediapb.NewMediaServiceClient(conn)
	return &MediaClient{client: client, conn: conn}, nil
}

func (c *MediaClient) Close() error {
	if c.conn != nil { return c.conn.Close() }
	return nil
}

func (c *MediaClient) UploadMedia(ctx context.Context, req *mediapb.UploadMediaRequest) (*mediapb.UploadMediaResponse, error) {
	return c.client.UploadMedia(ctx, req)
}

func (c *MediaClient) GetMediaMetadata(ctx context.Context, req *mediapb.GetMediaMetadataRequest) (*mediapb.Media, error) {
	return c.client.GetMediaMetadata(ctx, req)
}

func (c *MediaClient) GetMultipleMediaMetadata(ctx context.Context, req *mediapb.GetMultipleMediaMetadataRequest) (*mediapb.GetMultipleMediaMetadataResponse, error) {
	return c.client.GetMultipleMediaMetadata(ctx, req)
}