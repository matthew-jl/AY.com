package grpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	messagepb "github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/repository/postgres"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/websocket"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MessageHandler struct {
	messagepb.UnimplementedMessageServiceServer
	repo       *postgres.MessageRepository
	wsHub      *websocket.Hub
	userClient userpb.UserServiceClient 
	mediaClient mediapb.MediaServiceClient
}

func NewMessageHandler(repo *postgres.MessageRepository, hub *websocket.Hub, uc userpb.UserServiceClient, mc mediapb.MediaServiceClient) *MessageHandler {
	return &MessageHandler{repo: repo, wsHub: hub, userClient: uc, mediaClient: mc}
}

// HealthCheck (implement similar to other services)
func (h *MessageHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*messagepb.HealthResponse, error) {
    if err := h.repo.CheckHealth(ctx); err != nil { return &messagepb.HealthResponse{Status: "Message Service DEGRADED"}, nil }
    return &messagepb.HealthResponse{Status: "Message Service OK"}, nil
}


func (h *MessageHandler) GetOrCreateDirectChat(ctx context.Context, req *messagepb.GetOrCreateDirectChatRequest) (*messagepb.Chat, error) {
	log.Printf("GetOrCreateDirectChat request between User %d and User %d", req.UserId1, req.UserId2)
	if req.UserId1 == 0 || req.UserId2 == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Both user IDs are required")
	}
	if req.UserId1 == req.UserId2 {
		return nil, status.Errorf(codes.InvalidArgument, "Cannot create a direct chat with oneself")
	}

	dbChat, err := h.repo.GetOrCreateDirectChat(ctx, uint(req.UserId1), uint(req.UserId2))
	if err != nil {
		log.Printf("Error in GetOrCreateDirectChat repo call: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get or create direct chat")
	}
	if dbChat == nil { return nil, status.Errorf(codes.Internal, "Failed to get or create chat (nil result)") }

	var participantsDB []postgres.ChatParticipant
    h.repo.DB().WithContext(ctx).Where("chat_id = ?", dbChat.ID).Find(&participantsDB)

    chatPreview := &postgres.ChatPreview{Chat: *dbChat, Participants: participantsDB}
    // Last message would be null for newly created chat

	return mapDBChatToProto(ctx, h, chatPreview, req.UserId1)
}


func (h *MessageHandler) SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.Message, error) {
	log.Printf("SendMessage request to Chat %d by User %d", req.ChatId, req.SenderId)
	if req.ChatId == 0 || req.SenderId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Chat ID and Sender ID are required")
	}
	if req.Content == "" && len(req.MediaIds) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Message must have content or media")
	}

	// TODO: Validate if SenderID is a participant of ChatID

	dbMessage := &postgres.Message{
		ChatID:   uint(req.ChatId),
		SenderID: uint(req.SenderId),
		Content:  req.Content,
		MediaIDs: uint32SliceToInt64ArrayMsg(req.MediaIds), // Helper
		Type:     "text", // Default, determine based on content/media
	}
	if len(req.MediaIds) > 0 {
		dbMessage.Type = "media" // Or more specific based on first media item
	}


	err := h.repo.CreateMessage(ctx, dbMessage)
	if err != nil {
		log.Printf("Error creating message in repo: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to send message")
	}

	senderSummary, _ := h.fetchUserSummary(ctx, req.SenderId)
    mediaItemsMap, _ := h.hydrateMediaItems(ctx, req.MediaIds)

	protoMsg := mapDBMessageToProto(dbMessage, senderSummary, mediaItemsMap)

	go func(chatID uint, msgToBroadcast *messagepb.Message) {
		log.Printf("Attempting to broadcast message ID %d to chat %d via WebSocket", msgToBroadcast.Id, chatID)
		h.wsHub.BroadcastMessageToChat(chatID, msgToBroadcast)
	}(uint(req.ChatId), protoMsg)

	return protoMsg, nil
}

// GetMessages retrieves messages for a chat.
func (h *MessageHandler) GetMessages(ctx context.Context, req *messagepb.GetMessagesRequest) (*messagepb.GetMessagesResponse, error) {
	log.Printf("GetMessages request for Chat %d by User %d", req.ChatId, req.UserId)
	if req.ChatId == 0 || req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Chat ID and User ID are required")
	}
	// TODO: Validate req.UserId is a participant of req.ChatId

	limit, offset := getLimitOffsetMsg(req.Page, req.Limit)
	dbMessages, err := h.repo.GetMessagesForChat(ctx, uint(req.ChatId), limit, offset)
	if err != nil {
		log.Printf("Error fetching messages for chat %d: %v", req.ChatId, err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch messages")
	}

	// Batch fetch sender summaries and media for all messages
	senderIDsSet := make(map[uint32]bool)
    allMediaIDsSet := make(map[uint32]bool)
	for _, msg := range dbMessages {
		senderIDsSet[uint32(msg.SenderID)] = true
        for _, mediaID := range msg.MediaIDs {
            allMediaIDsSet[uint32(mediaID)] = true
        }
	}
	var senderIDsToFetch []uint32; for id := range senderIDsSet { senderIDsToFetch = append(senderIDsToFetch, id) }
    var mediaIDsToFetch []uint32; for id := range allMediaIDsSet { mediaIDsToFetch = append(mediaIDsToFetch, id) }


    sendersMap, _ := h.hydrateUserSummaries(ctx, senderIDsToFetch)
    mediaItemsMap, _ := h.hydrateMediaItems(ctx, mediaIDsToFetch)


	protoMessages := make([]*messagepb.Message, len(dbMessages))
	for i, dm := range dbMessages {
		protoMessages[i] = mapDBMessageToProto(&dm, sendersMap[uint32(dm.SenderID)], mediaItemsMap)
	}
	return &messagepb.GetMessagesResponse{Messages: protoMessages, HasMore: len(dbMessages) == limit}, nil
}

// DeleteMessage allows sender to delete their message within a time window.
func (h *MessageHandler) DeleteMessage(ctx context.Context, req *messagepb.DeleteMessageRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteMessage request: MessageID %d by User %d", req.MessageId, req.UserId)
	if req.MessageId == 0 || req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Message ID and User ID are required")
	}

	deletedMsg, err := h.repo.MarkMessageAsDeleted(ctx, uint(req.MessageId), uint(req.UserId))
	if err != nil {
		log.Printf("Error deleting message %d by user %d: %v", req.MessageId, req.UserId, err)
		if err.Error() == "message not found or not owned by user" {
			return nil, status.Errorf(codes.NotFound, "%s", err.Error())
		}
		if err.Error() == "message can no longer be deleted (past 1 minute window)" {
			return nil, status.Errorf(codes.FailedPrecondition, "%s", err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete message")
	}
	log.Printf("Message %d soft-deleted by user %d", req.MessageId, req.UserId)

	var chatIDForBroadcast uint32 = uint32(deletedMsg.ChatID)
	if chatIDForBroadcast != 0 { // Only broadcast if we know the chat
		updatePayload := websocket.MessageUpdateEvent{
			Type:       "message_deleted",
			ChatID:    chatIDForBroadcast,
			MessageID: req.MessageId,
			ActorID: req.UserId,
		}
        go h.wsHub.BroadcastMessageUpdateToChat(uint(chatIDForBroadcast), updatePayload)
	} else {
		log.Printf("Could not determine ChatID for deleted message %d to broadcast WS update.", req.MessageId)
	}

	return &emptypb.Empty{}, nil
}

// GetUserChats retrieves a list of chats for the user.
func (h *MessageHandler) GetUserChats(ctx context.Context, req *messagepb.GetUserChatsRequest) (*messagepb.GetUserChatsResponse, error) {
    log.Printf("GetUserChats request for User %d", req.UserId)
    if req.UserId == 0 { 
		log.Println("GetUserChats: Invalid UserId")
		return nil, status.Errorf(codes.InvalidArgument, "UserId is required")
		
	}
    limit, offset := getLimitOffsetMsg(req.Page, req.Limit)

    dbChatPreviews, err := h.repo.GetUserChats(ctx, uint(req.UserId), limit, offset)
    if err != nil { 
		log.Printf("Error fetching user chats: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch user chats") 
	}

    protoChats := make([]*messagepb.Chat, 0, len(dbChatPreviews))
    for i := range dbChatPreviews { // Iterate over index to use pointer
        mappedChat, errMap := mapDBChatToProto(ctx, h, &dbChatPreviews[i], req.UserId)
        if errMap != nil {
            log.Printf("Error mapping chat preview ID %d for GetUserChats: %v", dbChatPreviews[i].ID, errMap)
            continue // Skip this chat on error
        }
        protoChats = append(protoChats, mappedChat)
    }
    return &messagepb.GetUserChatsResponse{Chats: protoChats, HasMore: len(dbChatPreviews) == limit}, nil
}

func (h *MessageHandler) DeleteChat(ctx context.Context, req *messagepb.DeleteChatRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteChat request: ChatID %d by User %d", req.ChatId, req.UserId)
	if req.ChatId == 0 || req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Chat ID and User ID are required")
	}

	err := h.repo.HideChatForUser(ctx, uint(req.ChatId), uint(req.UserId))
	if err != nil {
		log.Printf("Error hiding chat %d for user %d: %v", req.ChatId, req.UserId, err)
		if err.Error() == "chat participation not found or already hidden" {
			return nil, status.Errorf(codes.NotFound, err.Error()) // Or just return OK if idempotent
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete/hide chat")
	}

	log.Printf("Chat %d hidden for user %d", req.ChatId, req.UserId)
	return &emptypb.Empty{}, nil
}

func (h *MessageHandler) CreateGroupChat(ctx context.Context, req *messagepb.CreateGroupChatRequest) (*messagepb.Chat, error) {
	log.Printf("CreateGroupChat request by User %d, Name: %s", req.CreatorId, req.GroupName)
	if req.CreatorId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Creator ID is required")
	}
	if strings.TrimSpace(req.GroupName) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Group name is required")
	}
	participantIDs := make([]uint, len(req.InitialParticipantIds))
	for i, pid := range req.InitialParticipantIds {
		participantIDs[i] = uint(pid)
	}

	dbChat, err := h.repo.CreateGroupChat(ctx, uint(req.CreatorId), req.GroupName, participantIDs)
	if err != nil {
		log.Printf("Error creating group chat in repo: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to create group chat: %v", err.Error())
	}
    if dbChat == nil { return nil, status.Errorf(codes.Internal, "Failed to create group chat (nil result)")}


    // Hydrate for response
    chatPreview := &postgres.ChatPreview{Chat: *dbChat, Participants: dbChat.Participants}
	return mapDBChatToProto(ctx, h, chatPreview, req.CreatorId)
}

func (h *MessageHandler) AddParticipantToGroup(ctx context.Context, req *messagepb.UpdateGroupParticipantsRequest) (*emptypb.Empty, error) {
	log.Printf("AddParticipantToGroup: Chat %d, Actor %d, TargetUser %d", req.ChatId, req.ActorUserId, req.TargetUserId)
	if req.ChatId == 0 || req.ActorUserId == 0 || req.TargetUserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Chat ID, Actor ID, and Target User ID are required")
	}
	// TODO: Permission check: Does ActorUserID have rights to add members to this ChatID?

	// Fetch chat to check if it's a group
	chat, err := h.repo.GetChatByID(ctx, uint(req.ChatId))
	if err != nil {
        if err.Error() == "chat not found" { return nil, status.Errorf(codes.NotFound, "Group chat not found")}
        return nil, status.Errorf(codes.Internal, "Failed to retrieve chat details")
    }
	if chat.Type != "group" { return nil, status.Errorf(codes.InvalidArgument, "Cannot add participant to a non-group chat")}


	added, err := h.repo.AddParticipantToGroup(ctx, uint(req.ChatId), uint(req.TargetUserId), uint(req.ActorUserId))
	if err != nil {
		log.Printf("Error adding participant to group: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to add participant: %v", err.Error())
	}

	if added {
		log.Printf("User %d added to group chat %d", req.TargetUserId, req.ChatId)
	}

	return &emptypb.Empty{}, nil
}

func (h *MessageHandler) RemoveParticipantFromGroup(ctx context.Context, req *messagepb.UpdateGroupParticipantsRequest) (*emptypb.Empty, error) {
	log.Printf("RemoveParticipantFromGroup: Chat %d, Actor %d, TargetUser %d", req.ChatId, req.ActorUserId, req.TargetUserId)
	if req.ChatId == 0 || req.ActorUserId == 0 || req.TargetUserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Chat ID, Actor ID, and Target User ID are required")
	}

	// TODO: Permission check: Does ActorUserID have rights (or is self-removal)?
	
	// Fetch chat
    chat, err := h.repo.GetChatByID(ctx, uint(req.ChatId))
	if err != nil {
        if err.Error() == "chat not found" { return nil, status.Errorf(codes.NotFound, "Group chat not found")}
        return nil, status.Errorf(codes.Internal, "Failed to retrieve chat details")
    }
	if chat.Type != "group" { return nil, status.Errorf(codes.InvalidArgument, "Cannot remove participant from a non-group chat")}

    // TODO: Prevent removing the creator if they are the last participant/admin (more complex logic needed)

	err = h.repo.RemoveParticipantFromGroup(ctx, uint(req.ChatId), uint(req.TargetUserId), uint(req.ActorUserId))
	if err != nil {
		log.Printf("Error removing participant from group: %v", err)
		if err.Error() == "participant not found in chat or already removed/hidden" {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Failed to remove participant: %v", err.Error())
	}

	log.Printf("User %d removed from group chat %d", req.TargetUserId, req.ChatId)
	return &emptypb.Empty{}, nil
}

// --- Helper functions ---

func (h *MessageHandler) fetchUserSummary(ctx context.Context, userID uint32) (*messagepb.UserSummary, error) {
    if userID == 0 || h.userClient == nil { return nil, errors.New("invalid user ID or user client not available")}
    profileResp, err := h.userClient.GetUserProfile(ctx, &userpb.GetUserProfileRequest{UserIdToView: userID})
    if err != nil { return nil, err }
    if profileResp == nil || profileResp.User == nil { return nil, errors.New("user profile not found for summary")}
    u := profileResp.User
    return &messagepb.UserSummary{Id: u.Id, Name: u.Name, Username: u.Username, ProfilePictureUrl: u.ProfilePicture}, nil
}

func (h *MessageHandler) hydrateUserSummaries(ctx context.Context, userIDs []uint32) (map[uint32]*messagepb.UserSummary, error) {
	if len(userIDs) == 0 || h.userClient == nil {
		return make(map[uint32]*messagepb.UserSummary), nil
	}
	profilesResp, err := h.userClient.GetUserProfilesByIds(ctx, &userpb.GetUserProfilesByIdsRequest{UserIds: userIDs})
	if err != nil { return nil, fmt.Errorf("failed to fetch user profiles for summaries: %w", err) }

	summaries := make(map[uint32]*messagepb.UserSummary)
	if profilesResp != nil && profilesResp.Users != nil {
		for uid, profile := range profilesResp.Users {
			if profile != nil {
				summaries[uid] = &messagepb.UserSummary{
					Id: uid, Name: profile.Name, Username: profile.Username, ProfilePictureUrl: profile.ProfilePicture,
				}
			}
		}
	}
	return summaries, nil
}

func (h *MessageHandler) hydrateMediaItems(ctx context.Context, mediaIDs []uint32) (map[uint32]*messagepb.Media, error) {
    if len(mediaIDs) == 0 || h.mediaClient == nil {
        return make(map[uint32]*messagepb.Media), nil
    }
    mediaResp, err := h.mediaClient.GetMultipleMediaMetadata(ctx, &mediapb.GetMultipleMediaMetadataRequest{MediaIds: mediaIDs})
    if err != nil { return nil, fmt.Errorf("failed to fetch media metadata: %w", err) }
    if mediaResp == nil || mediaResp.MediaItems == nil {
		return make(map[uint32]*messagepb.Media), nil
	}
    mediaItems := make(map[uint32]*messagepb.Media)
	for mediaID, media := range mediaResp.MediaItems {
		if media != nil {
			// Map mediapb.Media to messagepb.Media
			mediaItems[mediaID] = &messagepb.Media{
				Id:              media.Id,
				UploaderUserId:  media.UploaderUserId,
				SupabasePath:    media.SupabasePath,
				BucketName:      media.BucketName,
				MimeType:        media.MimeType,
				FileSize:        media.FileSize,
				PublicUrl:       media.PublicUrl,
				CreatedAt:       media.CreatedAt,
			}
		}
	}
	return mediaItems, nil
}


func mapDBMessageToProto(m *postgres.Message, senderSummary *messagepb.UserSummary, mediaItemsMap map[uint32]*messagepb.Media) *messagepb.Message {
	if m == nil { return nil }
	protoMsg := &messagepb.Message{
		Id:       uint32(m.ID), ChatId: uint32(m.ChatID), SenderId: uint32(m.SenderID),
		Content:  m.Content, Type: m.Type, // MediaIds will be populated by hydrated mediaItems
		SentAt:   timestamppb.New(m.SentAt), IsDeleted: m.IsDeleted,
		SenderSummary: senderSummary,
	}

    if len(m.MediaIDs) > 0 && mediaItemsMap != nil {
        protoMsg.MediaItems = make([]*messagepb.Media, 0, len(m.MediaIDs))
        for _, dbMediaID := range m.MediaIDs {
            if mediaItem, ok := mediaItemsMap[uint32(dbMediaID)]; ok && mediaItem != nil {
                protoMsg.MediaItems = append(protoMsg.MediaItems, mediaItem)
            }
        }
    }
	return protoMsg
}

// mapDBChatToProto (for GetOrCreateDirectChat and GetUserChats)
func mapDBChatToProto(ctx context.Context, h *MessageHandler, chatPreview *postgres.ChatPreview, currentUserID uint32) (*messagepb.Chat, error) {
    if chatPreview == nil { return nil, errors.New("nil chat preview") }

    // Hydrate participants
    participantIDs := make([]uint32, 0, len(chatPreview.Participants))
    for _, p := range chatPreview.Participants {
        participantIDs = append(participantIDs, uint32(p.UserID))
    }
    participantSummariesMap, err := h.hydrateUserSummaries(ctx, participantIDs)
    if err != nil { log.Printf("Error hydrating participants for chat %d: %v", chatPreview.ID, err) }

    mappedParticipants := make([]*messagepb.UserSummary, 0, len(participantIDs))
    for _, pid := range participantIDs {
        if summary, ok := participantSummariesMap[pid]; ok {
            mappedParticipants = append(mappedParticipants, summary)
        }
    }

    // Hydrate last message (if exists)
    var lastProtoMessage *messagepb.Message
    if chatPreview.LastMessage != nil && chatPreview.LastMessage.ID != 0 {
        var lastMsgSenderSummary *messagepb.UserSummary
        if summary, ok := participantSummariesMap[uint32(chatPreview.LastMessage.SenderID)]; ok {
            lastMsgSenderSummary = summary
        } else { // Fetch if not already fetched (e.g., sender not in current participant list)
            lastMsgSenderSummary, _ = h.fetchUserSummary(ctx, uint32(chatPreview.LastMessage.SenderID))
        }

        // Hydrate media for last message
        var lastMsgMediaMap map[uint32]*messagepb.Media
        if len(chatPreview.LastMessage.MediaIDs) > 0 {
            lastMsgMediaMap, _ = h.hydrateMediaItems(ctx, int64ArrayToUint32SliceMsg(chatPreview.LastMessage.MediaIDs))
        }
        lastProtoMessage = mapDBMessageToProto(chatPreview.LastMessage, lastMsgSenderSummary, lastMsgMediaMap)
    }

    chatName := ""
    if chatPreview.Type == "group" && chatPreview.Name != nil {
        chatName = *chatPreview.Name
    } else if chatPreview.Type == "direct" {
        // For direct chat, try to set name as the other user's name
        for _, p := range mappedParticipants {
            if p.Id != currentUserID {
                chatName = p.Name // Use actual name
                break
            }
        }
    }


    return &messagepb.Chat{
        Id:           uint32(chatPreview.ID),
        Type:         chatPreview.Type,
        Name:         &chatName, // Use derived name
        CreatorId:    uint32(chatPreview.CreatorID),
        CreatedAt:    timestamppb.New(chatPreview.CreatedAt),
        UpdatedAt:    timestamppb.New(chatPreview.UpdatedAt),
        Participants: mappedParticipants,
        LastMessage:  lastProtoMessage,
    }, nil
}

func uint32SliceToInt64ArrayMsg(s []uint32) pq.Int64Array {
	if s == nil {
		return pq.Int64Array{}
	}
	arr := make(pq.Int64Array, len(s))
	for i, v := range s {
		arr[i] = int64(v)
	}
	return arr
}
func int64ArrayToUint32SliceMsg(arr pq.Int64Array) []uint32 {
	if arr == nil {
		return nil
	}
	s := make([]uint32, len(arr))
	for i, v := range arr {
		s[i] = uint32(v)
	}
	return s
}
func getLimitOffsetMsg(page, limit int32) (int, int) {
	if limit <= 0 { limit = 20 }
	if page <= 0 { page = 1 }
	offset := (page - 1) * limit
	return int(limit), int(offset)
}