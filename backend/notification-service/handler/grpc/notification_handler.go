package grpc

import (
	"context"

	notifpb "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/repository/postgres"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NotificationHandler struct {
	notifpb.UnimplementedNotificationServiceServer
	repo *postgres.NotificationRepository
}

func NewNotificationHandler(repo *postgres.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

func (h *NotificationHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*notifpb.HealthResponse, error) {
	if err := h.repo.CheckHealth(ctx); err != nil {
		return &notifpb.HealthResponse{Status: "Notification Service DEGRADED"}, nil
	}
	return &notifpb.HealthResponse{Status: "Notification Service OK"}, nil
}

func (h *NotificationHandler) GetNotifications(ctx context.Context, req *notifpb.GetNotificationsRequest) (*notifpb.GetNotificationsResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "User ID required")
	}
	limit, offset := getLimitOffsetNotif(req.Page, req.Limit)

	dbNotifs, err := h.repo.GetNotificationsForUser(ctx, uint(req.UserId), limit, offset, req.UnreadOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get notifications")
	}

	protoNotifs := make([]*notifpb.Notification, len(dbNotifs))
	for i, dn := range dbNotifs {
		protoNotifs[i] = mapDBNotificationToProto(&dn)
	}
	return &notifpb.GetNotificationsResponse{Notifications: protoNotifs, HasMore: len(dbNotifs) == limit}, nil
}

func (h *NotificationHandler) MarkNotificationAsRead(ctx context.Context, req *notifpb.MarkNotificationAsReadRequest) (*emptypb.Empty, error) {
	if req.NotificationId == 0 || req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "IDs required")
	}
	err := h.repo.MarkNotificationAsRead(ctx, uint(req.NotificationId), uint(req.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to mark as read")
	}
	return &emptypb.Empty{}, nil
}

func (h *NotificationHandler) MarkAllNotificationsAsRead(ctx context.Context, req *notifpb.MarkAllNotificationsAsReadRequest) (*emptypb.Empty, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "User ID required")
	}
	err := h.repo.MarkAllNotificationsAsRead(ctx, uint(req.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to mark all as read")
	}
	return &emptypb.Empty{}, nil
}

func (h *NotificationHandler) GetUnreadNotificationCount(ctx context.Context, req *notifpb.GetUnreadNotificationCountRequest) (*notifpb.GetUnreadNotificationCountResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "User ID required")
	}
	count, err := h.repo.GetUnreadNotificationCount(ctx, uint(req.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get unread count")
	}
	return &notifpb.GetUnreadNotificationCountResponse{Count: int32(count)}, nil
}

func getLimitOffsetNotif(page, limit int32) (int, int) { 
	p := int(page); l := int(limit)
	if l <= 0 || l > 50 { l = 20 }
	if p <= 0 { p = 1 }
	return l, (p - 1) * l
}

func mapDBNotificationToProto(n *postgres.Notification) *notifpb.Notification {
	if n == nil {
		return nil
	}
	pn := notifpb.Notification{
		Id:        uint32(n.ID),
		UserId:    uint32(n.UserID),
		Type:      n.Type,
		Message:   n.Message,
		IsRead:    n.IsRead,
		EntityId:  n.EntityID,
		CreatedAt: timestamppb.New(n.CreatedAt),
	}
	if n.ActorID != nil {
		actorId := uint32(*n.ActorID)
		pn.ActorId = &actorId
	}
	return &pn
}