package http

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	threadpb "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/genproto/proto"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ThreadHandler struct {
	threadClient *client.ThreadClient
	mediaClient  *client.MediaClient
	userClient  *client.UserClient
}

func NewThreadHandler(threadClient *client.ThreadClient, mediaClient *client.MediaClient, userClient *client.UserClient) *ThreadHandler {
	return &ThreadHandler{threadClient: threadClient, mediaClient: mediaClient, userClient: userClient}
}

// Payload for creating a thread (matches frontend structure)
type CreateThreadPayload struct {
	Content          string   `json:"content"`
	ParentThreadID   *uint32  `json:"parent_thread_id,omitempty"`
	ReplyRestriction string   `json:"reply_restriction,omitempty"` // e.g., "EVERYONE", "FOLLOWING"
	ScheduledAt      *string  `json:"scheduled_at,omitempty"`
	CommunityID      *uint32  `json:"community_id,omitempty"`
	MediaIDs         []uint32 `json:"media_ids,omitempty"`
}

type FrontendMediaMetadata struct {
	ID             uint32 `json:"id"`
	UploaderUserID uint32 `json:"uploader_user_id"`
	SupabasePath   string `json:"supabase_path"`
	BucketName     string `json:"bucket_name"`
	MimeType       string `json:"mime_type"`
	FileSize       int64  `json:"file_size"`
	PublicURL      string `json:"public_url"`
	CreatedAt      string `json:"created_at"`
}

type FrontendUserProfile struct {
	ID             uint32 `json:"id"`
	Name           string `json:"name"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	ProfilePicture string `json:"profile_picture,omitempty"`
	// Add other fields frontend needs, e.g., account_privacy, is_verified
}

type FrontendThreadData struct {
	ID               uint32                `json:"id"`
	UserID           uint32                `json:"user_id"`
	Content          string                `json:"content"`
	ParentThreadID   *uint32               `json:"parent_thread_id,omitempty"`
	ReplyRestriction string                `json:"reply_restriction"`
	ScheduledAt      *string               `json:"scheduled_at,omitempty"`
	PostedAt         string                `json:"posted_at"`             
	CommunityID      *uint32               `json:"community_id,omitempty"`
	IsAdvertisement  bool                  `json:"is_advertisement"`
	MediaIDs         []uint32              `json:"media_ids"`
	CreatedAt        string                `json:"created_at"`     
	Author           *FrontendUserProfile  `json:"author,omitempty"`  // Hydrated
	Media            []FrontendMediaMetadata `json:"media,omitempty"`   // Hydrated
	LikeCount        int32                 `json:"like_count"`
	ReplyCount       int32                 `json:"reply_count"`
	RepostCount      int32                 `json:"repost_count"`
	BookmarkCount               int32                 `json:"bookmark_count"`
	IsLikedByCurrentUser        bool                  `json:"is_liked_by_current_user"`
	IsBookmarkedByCurrentUser   bool                  `json:"is_bookmarked_by_current_user"`
}

type FrontendFeedResponse struct {
	Threads []FrontendThreadData `json:"threads"`
	HasMore bool                 `json:"has_more"`
}



func (h *ThreadHandler) CreateThread(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return }

	var payload CreateThreadPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
		return
	}

	if payload.Content == "" && len(payload.MediaIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thread must contain content or media"})
		return
	}

	grpcReq := &threadpb.CreateThreadRequest{
		UserId:   userID,
		Content:  payload.Content,
		MediaIds: payload.MediaIDs,
	}

	grpcReq.ReplyRestriction = mapHTTPReplyRestrictionToProto(payload.ReplyRestriction)

	if payload.ParentThreadID != nil {
		grpcReq.ParentThreadId = payload.ParentThreadID
	}
	if payload.CommunityID != nil {
		grpcReq.CommunityId = payload.CommunityID
	}
	if payload.ScheduledAt != nil && *payload.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, *payload.ScheduledAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled_at format. Use ISO 8601 (RFC3339)."})
			return
		}
		grpcReq.ScheduledAt = timestamppb.New(t)
	}

	// Call Thread Service
	createdThread, err := h.threadClient.CreateThread(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "create thread", err)
		return
	}

	c.JSON(http.StatusCreated, createdThread)
}

func (h *ThreadHandler) GetThread(c *gin.Context) {
	currentUserID, _ := getUserIDFromContext(c)
	threadID, ok := getUint32Param(c, "threadId")
	if !ok { return }

	grpcReq := &threadpb.GetThreadRequest{ThreadId: threadID, CurrentUserId: &currentUserID}
	threadProto, err := h.threadClient.GetThread(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get thread", err)
		return
	}
	var authorsMap map[uint32]*userpb.User
	var mediaMap map[uint32]*mediapb.Media
	var userErr, mediaErr error // Not strictly needed if fetching for single thread without goroutines

	if threadProto.GetUserId() != 0 {
		// Fetch single author
		resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: []uint32{threadProto.GetUserId()}})
		if err == nil && resp != nil { authorsMap = resp.GetUsers() } else { userErr = err }
	}
	if len(threadProto.GetMediaIds()) > 0 {
		// Fetch multiple media
		resp, err := h.mediaClient.GetMultipleMediaMetadata(c.Request.Context(), &mediapb.GetMultipleMediaMetadataRequest{MediaIds: threadProto.GetMediaIds()})
		if err == nil && resp != nil { mediaMap = resp.GetMediaItems() } else { mediaErr = err }
	}
    if userErr != nil { log.Printf("Error fetching author for GetThread: %v", userErr) }
    if mediaErr != nil { log.Printf("Error fetching media for GetThread: %v", mediaErr) }


	feThread := mapProtoThreadToFrontend(threadProto, authorsMap, mediaMap)
	c.JSON(http.StatusOK, feThread)
}

func (h *ThreadHandler) DeleteThread(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return }
	threadID, ok := getUint32Param(c, "threadId")
	if !ok { return }

	grpcReq := &threadpb.DeleteThreadRequest{ThreadId: threadID, UserId: userID}
	_, err := h.threadClient.DeleteThread(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "delete thread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Thread deleted successfully"}) // Or 204 No Content
}

// --- Interaction Handlers ---

func (h *ThreadHandler) LikeThread(c *gin.Context) {
	h.handleInteraction(c, "like")
}

func (h *ThreadHandler) UnlikeThread(c *gin.Context) {
	h.handleInteraction(c, "unlike")
}

func (h *ThreadHandler) BookmarkThread(c *gin.Context) {
	h.handleInteraction(c, "bookmark")
}

func (h *ThreadHandler) UnbookmarkThread(c *gin.Context) {
	h.handleInteraction(c, "unbookmark")
}

func (h *ThreadHandler) handleInteraction(c *gin.Context, action string) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return }
	threadID, ok := getUint32Param(c, "threadId")
	if !ok { return }

	grpcReq := &threadpb.InteractThreadRequest{ThreadId: threadID, UserId: userID}
	var err error
	var operation string

	switch action {
	case "like":
		_, err = h.threadClient.LikeThread(c.Request.Context(), grpcReq)
		operation = "like thread"
	case "unlike":
		_, err = h.threadClient.UnlikeThread(c.Request.Context(), grpcReq)
		operation = "unlike thread"
	case "bookmark":
		_, err = h.threadClient.BookmarkThread(c.Request.Context(), grpcReq)
		operation = "bookmark thread"
	case "unbookmark":
		_, err = h.threadClient.UnbookmarkThread(c.Request.Context(), grpcReq)
		operation = "unbookmark thread"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interaction action"})
		return
	}

	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.AlreadyExists || st.Code() == codes.NotFound) {
            log.Printf("Idempotent %s action for thread %d user %d resulted in: %s", action, threadID, userID, st.Code())
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Interaction '%s' processed", action)})
			return
		}
		handleGRPCError(c, operation, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Successfully %sd thread", action)})
}

func (h *ThreadHandler) GetFeed(c *gin.Context) {
	currentUserID, _ := getUserIDFromContext(c)

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	feedType := c.DefaultQuery("type", "foryou") // Default to 'foryou'

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 { page = 1 }
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 { limit = 20 }


	grpcReq := &threadpb.GetFeedThreadsRequest{
		CurrentUserId:   &currentUserID,
		Page:     int32(page),
		Limit:    int32(limit),
		FeedType: feedType,
	}

	// 1. Fetch base threads from Thread Service
	threadServiceResp, err := h.threadClient.GetFeedThreads(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get feed threads", err)
		return
	}

	if len(threadServiceResp.GetThreads()) == 0 {
		c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false})
		return
	}

	// 2. Collect User IDs and Media IDs for batch fetching
	authorIDsSet := make(map[uint32]bool)
	mediaIDsSet := make(map[uint32]bool)
	for _, t := range threadServiceResp.GetThreads() {
		if t.GetUserId() != 0 {
			authorIDsSet[t.GetUserId()] = true
		}
		for _, mediaID := range t.GetMediaIds() {
			if mediaID != 0 {
				mediaIDsSet[mediaID] = true
			}
		}
	}

	var authorIDs []uint32
	for id := range authorIDsSet { authorIDs = append(authorIDs, id) }
	var mediaIDs []uint32
	for id := range mediaIDsSet { mediaIDs = append(mediaIDs, id) }

	// 3. Fetch Author and Media data in parallel
	var wg sync.WaitGroup
	var authorsMap map[uint32]*userpb.User
	var mediaMap map[uint32]*mediapb.Media
	var userErr, mediaErr error

	if len(authorIDs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: authorIDs})
			if err != nil {
				userErr = err
				return
			}
			authorsMap = resp.GetUsers()
		}()
	}

	if len(mediaIDs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := h.mediaClient.GetMultipleMediaMetadata(c.Request.Context(), &mediapb.GetMultipleMediaMetadataRequest{MediaIds: mediaIDs})
			if err != nil {
				mediaErr = err
				return
			}
			mediaMap = resp.GetMediaItems()
		}()
	}
	wg.Wait() // Wait for both goroutines to finish

	// For now, no error handling for empty authors or media
	if userErr != nil {
		log.Printf("Error fetching author profiles: %v", userErr)
	}
	if mediaErr != nil {
		log.Printf("Error fetching media metadata: %v", mediaErr)
	}


	// 4. Hydrate Threads
	hydratedThreads := make([]FrontendThreadData, 0, len(threadServiceResp.GetThreads()))
	for _, tProto := range threadServiceResp.GetThreads() {
		feThread := mapProtoThreadToFrontend(tProto, authorsMap, mediaMap)
		hydratedThreads = append(hydratedThreads, feThread)
	}

	c.JSON(http.StatusOK, FrontendFeedResponse{
		Threads: hydratedThreads,
		HasMore: threadServiceResp.GetHasMore(),
	})
}

// --- Helper Functions ---

func getUserIDFromContext(c *gin.Context) (uint32, bool) {
	userIDAny, exists := c.Get("userID")
	if !exists {
		log.Println("ERROR: userID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication context missing"})
		return 0, false
	}
	userID, ok := userIDAny.(uint) // Comes as uint from middleware
	if !ok || userID == 0 {
		log.Printf("ERROR: Invalid userID type or value in context: %v (%T)", userIDAny, userIDAny)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid authentication context"})
		return 0, false
	}
	return uint32(userID), true
}

func getUint32Param(c *gin.Context, paramName string) (uint32, bool) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Printf("ERROR: Invalid %s parameter: %s", paramName, idStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid %s parameter", paramName)})
		return 0, false
	}
	return uint32(id), true
}

func mapHTTPReplyRestrictionToProto(s string) threadpb.ReplyRestriction {
	switch strings.ToUpper(s) {
	case "FOLLOWING": return threadpb.ReplyRestriction_FOLLOWING
	case "VERIFIED": return threadpb.ReplyRestriction_VERIFIED
	case "EVERYONE": fallthrough
	default:         return threadpb.ReplyRestriction_EVERYONE
	}
}

func handleGRPCError(c *gin.Context, operation string, err error) {
	st, ok := status.FromError(err)
	log.Printf("gRPC error during '%s': Code=%s, Msg=%s", operation, st.Code(), st.Message())
	if ok {
		httpCode := grpcStatusCodeToHTTP(st.Code())
		c.JSON(httpCode, gin.H{"error": st.Message()})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to %s: %s", operation, err.Error())})
	}
}

func mapProtoThreadToFrontend(tProto *threadpb.Thread, authorsMap map[uint32]*userpb.User, mediaMap map[uint32]*mediapb.Media) FrontendThreadData {
	feThread := FrontendThreadData{
		ID:                          tProto.GetId(),
		UserID:                      tProto.GetUserId(),
		Content:                     tProto.GetContent(),
		ReplyRestriction:            tProto.GetReplyRestriction().String(),
		PostedAt:                    tProto.GetPostedAt().AsTime().Format(time.RFC3339),
		IsAdvertisement:             tProto.GetIsAdvertisement(),
		MediaIDs:                    tProto.GetMediaIds(),
		CreatedAt:                   tProto.GetCreatedAt().AsTime().Format(time.RFC3339),
		LikeCount:                   tProto.GetLikeCount(),
		ReplyCount:                  tProto.GetReplyCount(),
		RepostCount:                 tProto.GetRepostCount(),
		BookmarkCount:               tProto.GetBookmarkCount(), // Added
		IsLikedByCurrentUser:        tProto.GetIsLikedByCurrentUser(),
		IsBookmarkedByCurrentUser:   tProto.GetIsBookmarkedByCurrentUser(),
	}
	if tProto.ParentThreadId != nil { val := tProto.GetParentThreadId(); feThread.ParentThreadID = &val }
	if tProto.CommunityId != nil { val := tProto.GetCommunityId(); feThread.CommunityID = &val }
	if tProto.GetScheduledAt().IsValid() { val := tProto.GetScheduledAt().AsTime().Format(time.RFC3339); feThread.ScheduledAt = &val }

	if authorsMap != nil {
		if authorProto, ok := authorsMap[tProto.GetUserId()]; ok && authorProto != nil {
			feThread.Author = &FrontendUserProfile{
				ID: authorProto.GetId(), Name: authorProto.GetName(), Username: authorProto.GetUsername(),
				Email: authorProto.GetEmail(), ProfilePicture: authorProto.GetProfilePicture(),
			}
		}
	}

	if mediaMap != nil && len(tProto.GetMediaIds()) > 0 {
		feThread.Media = make([]FrontendMediaMetadata, 0)
		for _, mediaID := range tProto.GetMediaIds() {
			if mediaProto, ok := mediaMap[mediaID]; ok && mediaProto != nil {
				feThread.Media = append(feThread.Media, FrontendMediaMetadata{
					ID: mediaProto.GetId(), UploaderUserID: mediaProto.GetUploaderUserId(),
					SupabasePath: mediaProto.GetSupabasePath(), BucketName: mediaProto.GetBucketName(),
					MimeType: mediaProto.GetMimeType(), FileSize: mediaProto.GetFileSize(),
					PublicURL: mediaProto.GetPublicUrl(), CreatedAt: mediaProto.GetCreatedAt().AsTime().Format(time.RFC3339),
				})
			}
		}
	}
	return feThread
}