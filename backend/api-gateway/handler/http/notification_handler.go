package http

import (
	"net/http"
	"strconv"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	notificationpb "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/genproto/proto"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationClient *client.NotificationClient
}

func NewNotificationHandler(nc *client.NotificationClient) *NotificationHandler {
	return &NotificationHandler{notificationClient: nc}
}

func (h *NotificationHandler) GetNotificationsHTTP(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return }

	page, limit := parsePagination(c)
	unreadOnlyStr := c.DefaultQuery("unread_only", "false")
	unreadOnly, _ := strconv.ParseBool(unreadOnlyStr)

	grpcReq := notificationpb.GetNotificationsRequest{
		UserId:     userID,
		Page:       page,
		Limit:      limit,
		UnreadOnly: unreadOnly,
	}
	resp, err := h.notificationClient.GetNotifications(c.Request.Context(), &grpcReq)
	if err != nil { handleGRPCError(c, "get notifications", err); return }
	c.JSON(http.StatusOK, resp)
}

// MarkAsReadHTTP marks a single notification as read
func (h *NotificationHandler) MarkAsReadHTTP(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return }

	notifIDStr := c.Param("notificationId")
	notifID, err := strconv.ParseUint(notifIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID format"}); return
	}

	grpcReq := notificationpb.MarkNotificationAsReadRequest{
		NotificationId: uint32(notifID),
		UserId:         userID, // Ensure user owns it
	}
	_, err = h.notificationClient.MarkNotificationAsRead(c.Request.Context(), &grpcReq)
	if err != nil { handleGRPCError(c, "mark notification as read", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func (h *NotificationHandler) MarkAllAsReadHTTP(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return }

	grpcReq := notificationpb.MarkAllNotificationsAsReadRequest{UserId: userID}
	_, err := h.notificationClient.MarkAllNotificationsAsRead(c.Request.Context(), &grpcReq)
	if err != nil { handleGRPCError(c, "mark all notifications as read", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

func (h *NotificationHandler) GetUnreadCountHTTP(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return }

	grpcReq := notificationpb.GetUnreadNotificationCountRequest{UserId: userID}
	resp, err := h.notificationClient.GetUnreadNotificationCount(c.Request.Context(), &grpcReq)
	if err != nil { handleGRPCError(c, "get unread notification count", err); return }
	c.JSON(http.StatusOK, resp) // Returns {"count": X}
}

func (h *NotificationHandler) NotificationServiceHealthHTTP(c *gin.Context) {
	resp, err := h.notificationClient.HealthCheck(c.Request.Context())
	if err != nil { handleGRPCError(c, "notification service health check", err); return }
	c.JSON(http.StatusOK, resp)
}