package client

import (
	"context"
	"fmt"

	notificationpb "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type NotificationClient struct {
	client notificationpb.NotificationServiceClient
	conn   *grpc.ClientConn
}

func NewNotificationClient(addr string) (*NotificationClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial notification service at %s: %w", addr, err)
	}
	client := notificationpb.NewNotificationServiceClient(conn)
	return &NotificationClient{client: client, conn: conn}, nil
}

func (c *NotificationClient) Close() error {
	if c.conn != nil { return c.conn.Close() }
	return nil
}

func (c *NotificationClient) GetNotifications(ctx context.Context, req *notificationpb.GetNotificationsRequest) (*notificationpb.GetNotificationsResponse, error) {
	return c.client.GetNotifications(ctx, req)
}

func (c *NotificationClient) MarkNotificationAsRead(ctx context.Context, req *notificationpb.MarkNotificationAsReadRequest) (*emptypb.Empty, error) {
	return c.client.MarkNotificationAsRead(ctx, req)
}

func (c *NotificationClient) MarkAllNotificationsAsRead(ctx context.Context, req *notificationpb.MarkAllNotificationsAsReadRequest) (*emptypb.Empty, error) {
	return c.client.MarkAllNotificationsAsRead(ctx, req)
}

func (c *NotificationClient) GetUnreadNotificationCount(ctx context.Context, req *notificationpb.GetUnreadNotificationCountRequest) (*notificationpb.GetUnreadNotificationCountResponse, error) {
    return c.client.GetUnreadNotificationCount(ctx, req)
}

func (c *NotificationClient) HealthCheck(ctx context.Context) (*notificationpb.HealthResponse, error) {
    return c.client.HealthCheck(ctx, &emptypb.Empty{})
}