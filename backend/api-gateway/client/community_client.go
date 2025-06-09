// backend/api-gateway/client/community_client.go
package client

import (
	"context"
	"fmt"

	communitypb "github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	// emptypb "google.golang.org/protobuf/types/known/emptypb" // If other RPCs return Empty
)

type CommunityClient struct {
	client communitypb.CommunityServiceClient
	conn   *grpc.ClientConn
}

func NewCommunityClient(addr string) (*CommunityClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial community service at %s: %w", addr, err)
	}
	client := communitypb.NewCommunityServiceClient(conn)
	return &CommunityClient{client: client, conn: conn}, nil
}

func (c *CommunityClient) Close() error {
	if c.conn != nil { return c.conn.Close() }
	return nil
}

func (c *CommunityClient) HealthCheck(ctx context.Context) (*communitypb.HealthResponse, error) {
    if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized") }
    return c.client.HealthCheck(ctx, &emptypb.Empty{})
}

func (c *CommunityClient) CreateCommunity(ctx context.Context, req *communitypb.CreateCommunityRequest) (*communitypb.CommunityDetails, error) {
	if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized for CommunityService") }
	return c.client.CreateCommunity(ctx, req)
}

func (c *CommunityClient) GetCommunityDetails(ctx context.Context, req *communitypb.GetCommunityDetailsRequest) (*communitypb.CommunityDetailsResponse, error) {
	if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized") }
	return c.client.GetCommunityDetails(ctx, req)
}

func (c *CommunityClient) ListCommunities(ctx context.Context, req *communitypb.ListCommunitiesRequest) (*communitypb.ListCommunitiesResponse, error) {
	if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized") }
	return c.client.ListCommunities(ctx, req)
}

func (c *CommunityClient) RequestToJoinCommunity(ctx context.Context, req *communitypb.CommunityUserRequest) (*emptypb.Empty, error) {
	if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized") }
	return c.client.RequestToJoinCommunity(ctx, req)
}

func (c *CommunityClient) AcceptJoinRequest(ctx context.Context, req *communitypb.CommunityUserActionRequest) (*emptypb.Empty, error) {
	if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized") }
	return c.client.AcceptJoinRequest(ctx, req)
}

func (c *CommunityClient) RejectJoinRequest(ctx context.Context, req *communitypb.CommunityUserActionRequest) (*emptypb.Empty, error) {
	if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized") }
	return c.client.RejectJoinRequest(ctx, req)
}

func (c *CommunityClient) GetCommunityMembers(ctx context.Context, req *communitypb.GetCommunityMembersRequest) (*communitypb.GetCommunityMembersResponse, error) {
	if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized") }
	return c.client.GetCommunityMembers(ctx, req)
}

func (c *CommunityClient) GetUserJoinRequests(ctx context.Context, req *communitypb.GetUserJoinRequestsRequest) (*communitypb.GetUserJoinRequestsResponse, error) {
    if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized")}
    return c.client.GetUserJoinRequests(ctx, req)
}

func (c *CommunityClient) GetCommunityPendingRequests(ctx context.Context, req *communitypb.GetCommunityPendingRequestsRequest) (*communitypb.GetCommunityPendingRequestsResponse, error) {
    if c.client == nil { return nil, fmt.Errorf("gRPC client not initialized")}
    return c.client.GetCommunityPendingRequests(ctx, req)
}