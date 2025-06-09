package http

import (
	"net/http"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	messagepb "github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/genproto/proto"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageClient *client.MessageClient
	userClient    *client.UserClient
    mediaClient   *client.MediaClient
}

// NewMessageHandler initializes a new message handler
func NewMessageHandler(mc *client.MessageClient, uc *client.UserClient, mediaC *client.MediaClient) *MessageHandler {
	return &MessageHandler{messageClient: mc, userClient: uc, mediaClient: mediaC}
}

// --- Payloads for HTTP requests ---
type GetOrCreateDirectChatPayload struct {
	OtherUserID uint32 `json:"other_user_id" binding:"required"`
}
type SendMessagePayload struct {
	Content  string   `json:"content"`
	MediaIDs []uint32 `json:"media_ids,omitempty"`
}

type CreateGroupChatPayload struct {
	Name                  string   `json:"name" binding:"required"`
	InitialParticipantIDs []uint32 `json:"initial_participant_ids"` // Can be empty
}
type UpdateGroupParticipantPayload struct {
    TargetUserID uint32 `json:"target_user_id" binding:"required"`
}

// GetOrCreateDirectChatHTTP creates or retrieves a direct chat
func (h *MessageHandler) GetOrCreateDirectChatHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }

	var payload GetOrCreateDirectChatPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()}); return
	}
	if payload.OtherUserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Other user ID is required"}); return
	}

	grpcReq := &messagepb.GetOrCreateDirectChatRequest{
		UserId1: requesterUserID,
		UserId2: payload.OtherUserID,
	}
	chat, err := h.messageClient.GetOrCreateDirectChat(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "get or create direct chat", err); return }

	c.JSON(http.StatusOK, chat)
}

// SendMessageHTTP sends a message to a specific chat
func (h *MessageHandler) SendMessageHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	chatID, ok := getUint32Param(c, "chatId")
	if !ok { return }

	var payload SendMessagePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message payload: " + err.Error()}); return
	}
	if payload.Content == "" && (payload.MediaIDs == nil || len(payload.MediaIDs) == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message must have content or media"}); return
	}

	grpcReq := &messagepb.SendMessageRequest{
		ChatId:   chatID,
		SenderId: requesterUserID,
		Content:  payload.Content,
		MediaIds: payload.MediaIDs,
	}
	sentMessage, err := h.messageClient.SendMessage(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "send message", err); return }

	c.JSON(http.StatusCreated, sentMessage)
}

// GetMessagesHTTP retrieves messages for a chat
func (h *MessageHandler) GetMessagesHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	chatID, ok := getUint32Param(c, "chatId")
	if !ok { return }

	page, limit := parsePagination(c)

	grpcReq := &messagepb.GetMessagesRequest{
		ChatId: chatID,
		UserId: requesterUserID, // For auth check within message service
		Page:   page,
		Limit:  limit,
	}
	messagesResp, err := h.messageClient.GetMessages(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "get messages", err); return }

	c.JSON(http.StatusOK, messagesResp)
}

// DeleteMessageHTTP deletes a message
func (h *MessageHandler) DeleteMessageHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	// chatId := c.Param("chatId") // Not strictly needed for delete if messageId is global
	messageID, ok := getUint32Param(c, "messageId")
	if !ok { return }

	grpcReq := &messagepb.DeleteMessageRequest{
		MessageId: messageID,
		UserId:    requesterUserID, // User trying to delete
	}
	_, err := h.messageClient.DeleteMessage(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "delete message", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

// GetUserChatsHTTP retrieves all chats for the authenticated user
func (h *MessageHandler) GetUserChatsHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	page, limit := parsePagination(c)

	grpcReq := &messagepb.GetUserChatsRequest{
		UserId: requesterUserID,
		Page:   page,
		Limit:  limit,
	}
	chatsResp, err := h.messageClient.GetUserChats(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "get user chats", err); return }

	c.JSON(http.StatusOK, chatsResp)
}

func (h *MessageHandler) DeleteChatHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	chatID, ok := getUint32Param(c, "chatId")
	if !ok { return }

	grpcReq := &messagepb.DeleteChatRequest{
		ChatId: chatID,
		UserId: requesterUserID,
	}
	_, err := h.messageClient.DeleteChat(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "delete chat", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Chat hidden successfully"})
}

func (h *MessageHandler) CreateGroupChatHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }

	var payload CreateGroupChatPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group chat payload: " + err.Error()}); return
	}

	grpcReq := &messagepb.CreateGroupChatRequest{
		CreatorId:            requesterUserID,
		GroupName:            payload.Name,
		InitialParticipantIds: payload.InitialParticipantIDs,
	}
	createdChat, err := h.messageClient.CreateGroupChat(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "create group chat", err); return }
	c.JSON(http.StatusCreated, createdChat) // Returns hydrated chat from message service
}

func (h *MessageHandler) AddParticipantHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	chatID, ok := getUint32Param(c, "chatId")
	if !ok { return }

	var payload UpdateGroupParticipantPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()}); return
	}

	grpcReq := &messagepb.UpdateGroupParticipantsRequest{
		ChatId:       chatID,
		ActorUserId:  requesterUserID,
		TargetUserId: payload.TargetUserID,
	}
	_, err := h.messageClient.AddParticipantToGroup(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "add participant to group", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Participant added successfully"})
}

func (h *MessageHandler) RemoveParticipantHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	chatID, ok := getUint32Param(c, "chatId")
	if !ok { return }
	targetUserID, ok := getUint32Param(c, "userId") // Get target user ID from path
	if !ok { return }

	grpcReq := &messagepb.UpdateGroupParticipantsRequest{
		ChatId:       chatID,
		ActorUserId:  requesterUserID,
		TargetUserId: targetUserID,
	}
	_, err := h.messageClient.RemoveParticipantFromGroup(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "remove participant from group", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Participant removed successfully"})
}

// MessageServiceHealthHTTP
func (h *MessageHandler) MessageServiceHealthHTTP(c *gin.Context) {
    resp, err := h.messageClient.HealthCheck(c.Request.Context())
    if err != nil {handleGRPCError(c, "message service health", err); return }
    c.JSON(http.StatusOK, resp)
}