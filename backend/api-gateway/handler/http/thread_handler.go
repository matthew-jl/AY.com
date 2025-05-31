package http

import (
	"context"
	"errors"
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
	AccountPrivacy string `json:"account_privacy,omitempty"`
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
	requesterUserID, _ := getUserIDFromContext(c)

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	feedType := c.DefaultQuery("type", "foryou") // Default to 'foryou'

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 { page = 1 }
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 { limit = 20 }

	// 1. Get IDs to exclude (blocked/blocking interactions relevant to the requester)
	excludeUserIDs, err := h.getFeedExclusionIDs(c.Request.Context(), requesterUserID)
	if err != nil {
		log.Printf("GetFeed: Error getting exclusion IDs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare feed filters"})
		return
	}

	// 2. Get IDs to include (only if "following" feed and requester is authenticated)
	var includeOnlyUserIDs []uint32
	if feedType == "following" {
		if requesterUserID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Must be logged in to view the 'following' feed"})
			return
		}
		followingResp, err := h.userClient.GetFollowingIDs(c.Request.Context(), &userpb.SocialListRequest{UserId: requesterUserID, Limit: 10000}) // Fetch all for now
		if err != nil {
			log.Printf("GetFeed: Error getting following IDs for user %d: %v", requesterUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load following feed data"})
			return
		}
		if followingResp != nil {
			includeOnlyUserIDs = followingResp.GetUserIds()
		}
		if len(includeOnlyUserIDs) == 0 { // User follows no one
			c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false})
			return
		}
	}


	grpcReq := &threadpb.GetFeedThreadsRequest{
		CurrentUserId:   &requesterUserID,
		Page:     int32(page),
		Limit:    int32(limit),
		FeedType: feedType,
		ExcludeUserIds:     excludeUserIDs,
		IncludeOnlyUserIds: includeOnlyUserIDs,
	}

	// 3. Fetch base threads from Thread Service
	threadServiceResp, err := h.threadClient.GetFeedThreads(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get feed threads", err)
		return
	}

	if len(threadServiceResp.GetThreads()) == 0 {
		c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false})
		return
	}

	// 4. Collect User IDs and Media IDs for batch fetching
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

	// 5. Fetch Author and Media data in parallel
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
	wg.Wait()

	if userErr != nil {
		log.Printf("Error fetching author profiles: %v", userErr)
	}
	if mediaErr != nil {
		log.Printf("Error fetching media metadata: %v", mediaErr)
	}

	// 6. Hydrate Threads
	hydratedThreads := make([]FrontendThreadData, 0, len(threadServiceResp.GetThreads()))
	for _, tProto := range threadServiceResp.GetThreads() {
		feThread := mapProtoThreadToFrontend(tProto, authorsMap, mediaMap)
		hydratedThreads = append(hydratedThreads, feThread)
	}

	// 7. Apply Privacy Filtering
	finalFilteredThreads := []FrontendThreadData{}
	if requesterUserID != 0 { // Authenticated user: apply complex privacy checks
		authorsToCheck := make(map[uint32]bool)
		for _, ht := range hydratedThreads {
			if ht.Author != nil && ht.Author.ID != requesterUserID { // Skip own threads
				authorsToCheck[ht.Author.ID] = true
			}
		}
		var authorIDsForPrivacyCheck []uint32
		for id := range authorsToCheck {
			authorIDsForPrivacyCheck = append(authorIDsForPrivacyCheck, id)
		}

		authorProfilesMap := make(map[uint32]*userpb.User)
		followStatusMap := make(map[uint32]bool)

		var privacyWg sync.WaitGroup
		var privacyUserErr, privacyFollowErr error

		if len(authorIDsForPrivacyCheck) > 0 {
			privacyWg.Add(1)
			go func() {
				defer privacyWg.Done()
				resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: authorIDsForPrivacyCheck})
				if err != nil {
					privacyUserErr = err
					return
				}
				if resp != nil {
					authorProfilesMap = resp.GetUsers()
				}
			}()

			privacyWg.Add(1)
			go func() {
				defer privacyWg.Done()
				for _, authorID := range authorIDsForPrivacyCheck {
					resp, err := h.userClient.IsFollowing(c.Request.Context(), &userpb.FollowCheckRequest{FollowerId: requesterUserID, FollowedId: authorID})
					if err == nil && resp != nil {
						followStatusMap[authorID] = resp.IsTrue
					}
				}
			}()
		}
		privacyWg.Wait()

		if privacyUserErr != nil {
			log.Printf("Error fetching author profiles for privacy filter: %v", privacyUserErr)
		}
		if privacyFollowErr != nil {
			log.Printf("Error fetching follow statuses for privacy filter: %v", privacyFollowErr)
		}

		for _, ht := range hydratedThreads {
			canView := true
			if ht.Author != nil && ht.Author.ID != requesterUserID { // Not the requester's own thread
				authorProfile, profileOk := authorProfilesMap[ht.Author.ID]
				if profileOk && authorProfile.GetAccountPrivacy() == "private" {
					isFollowing, followOk := followStatusMap[ht.Author.ID]
					if !followOk || !isFollowing {
						canView = false
					}
				}
			}
			if canView {
				finalFilteredThreads = append(finalFilteredThreads, ht)
			}
		}
	} else { // Unauthenticated user: only show public threads
		for _, ht := range hydratedThreads {
			if ht.Author != nil && ht.Author.AccountPrivacy == "public" {
				finalFilteredThreads = append(finalFilteredThreads, ht)
			}
		}
	}

	c.JSON(http.StatusOK, FrontendFeedResponse{
		Threads: hydratedThreads,
		HasMore: threadServiceResp.GetHasMore(),
	})
}

func (h *ThreadHandler) GetUserSpecificThreads(c *gin.Context) {
	usernameToView := c.Param("username")
	if usernameToView == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username parameter is required"})
		return
	}
	userResp, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: usernameToView})
	if err != nil {
		handleGRPCError(c, "resolve username for user threads", err)
		return
	}
	targetUserID := userResp.GetId()
	if targetUserID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	requesterUserID, _ := getUserIDFromContext(c)

	page, limit := parsePagination(c)
	threadType := c.DefaultQuery("type", "posts") // "posts", "replies", "likes", "media"

	// Validate threadType
	allowedTypes := map[string]bool{"posts": true, "replies": true, "likes": true, "media": true}
	if !allowedTypes[threadType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid thread type specified"})
		return
	}

	// 2. Perform Block and Privacy Checks (critical before fetching target's threads)
	if requesterUserID != 0 && requesterUserID != targetUserID { // Only if viewing someone else's profile while logged in
		var isBlockedByTargetStatus, hasRequesterBlockedTargetStatus bool
		var errBlock1, errBlock2 error
        var wgBlock sync.WaitGroup // Check blocks in parallel

        wgBlock.Add(2)
		go func() {
            defer wgBlock.Done()
			resp, err := h.userClient.IsBlockedBy(c.Request.Context(), &userpb.BlockCheckRequest{ActorId: requesterUserID, SubjectId: targetUserID})
			if err != nil { errBlock1 = err; return }
			if resp != nil { isBlockedByTargetStatus = resp.IsTrue }
		}()
		go func() {
            defer wgBlock.Done()
			resp, err := h.userClient.HasBlocked(c.Request.Context(), &userpb.BlockCheckRequest{ActorId: requesterUserID, SubjectId: targetUserID})
			if err != nil { errBlock2 = err; return }
			if resp != nil { hasRequesterBlockedTargetStatus = resp.IsTrue }
		}()
        wgBlock.Wait()

        if errBlock1 != nil { log.Printf("Error checking IsBlockedBy: %v", errBlock1) /* handle error, maybe proceed cautiously */ }
        if errBlock2 != nil { log.Printf("Error checking HasBlocked: %v", errBlock2) /* handle error */ }

		if isBlockedByTargetStatus || hasRequesterBlockedTargetStatus {
			log.Printf("Access to %s's %s denied for user %d due to blocking.", usernameToView, threadType, requesterUserID)
			c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false}) // Return empty due to block
			return
		}

		// Check target user's privacy setting against follow status
		targetProfileResp, err := h.userClient.GetUserProfile(c.Request.Context(), &userpb.GetUserProfileRequest{
			UserIdToView:    targetUserID,
			RequesterUserId: &requesterUserID,
		})
		if err == nil && targetProfileResp != nil && targetProfileResp.User != nil {
			if targetProfileResp.User.AccountPrivacy == "private" && !targetProfileResp.IsFollowedByRequester {
				log.Printf("Access to %s's %s denied for user %d due to private profile not followed.", usernameToView, threadType, requesterUserID)
				c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false})
				return
			}
		} else if err != nil {
			log.Printf("Error checking target profile privacy for user threads: %v", err)
			// Potentially fail here if privacy is strict
		}
	} else if requesterUserID == 0 && targetUserID != 0 { // Unauthenticated user viewing a profile
        // Check target user's privacy setting (can only see public)
         targetProfileResp, err := h.userClient.GetUserProfile(c.Request.Context(), &userpb.GetUserProfileRequest{UserIdToView: targetUserID /* Requester ID is 0 */})
         if err == nil && targetProfileResp != nil && targetProfileResp.User != nil {
             if targetProfileResp.User.AccountPrivacy == "private" {
                 log.Printf("Unauthenticated access to private profile threads of %s denied.", usernameToView)
                 c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false})
                 return
             }
         } else if err != nil {
              log.Printf("Error checking target profile privacy for unauthenticated user threads: %v", err)
         }
    }

	// 3. Get Exclude IDs (users blocked by/blocking the requester) for filtering the target's content.
	excludeUserIDs, err := h.getFeedExclusionIDs(c.Request.Context(), requesterUserID)
	if err != nil {
		log.Printf("GetUserSpecificThreads: Error getting exclusion IDs: %v", err)
	}

	grpcReq := &threadpb.GetUserThreadsRequest{
		TargetUserId:   targetUserID,
		RequesterUserId: &requesterUserID,
		ThreadType:     threadType,
		Page:           page,
		Limit:          limit,
		ExcludeUserIds: excludeUserIDs,
	}

	threadServiceResp, err := h.threadClient.GetUserThreads(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, fmt.Sprintf("get user %s threads", threadType), err)
		return
	}

	// Hydrate Author and Media for the fetched threads
	if len(threadServiceResp.GetThreads()) == 0 {
		c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false}) // Use same FrontendFeedResponse
		return
	}
	authorIDsSet := make(map[uint32]bool)
	mediaIDsSet := make(map[uint32]bool)
	for _, t := range threadServiceResp.GetThreads() {
		if t.GetUserId() != 0 { authorIDsSet[t.GetUserId()] = true }
		for _, mediaID := range t.GetMediaIds() { if mediaID != 0 { mediaIDsSet[mediaID] = true } }
	}
	var authorIDs []uint32; for id := range authorIDsSet { authorIDs = append(authorIDs, id) }
	var mediaIDs []uint32; for id := range mediaIDsSet { mediaIDs = append(mediaIDs, id) }

	var wg sync.WaitGroup
	var authorsMap map[uint32]*userpb.User
	var mediaMap map[uint32]*mediapb.Media
	var userErr, mediaErr error

	if len(authorIDs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if h.userClient == nil {
				log.Println("GetUserSpecificThreads: userClient in ThreadHandler is nil")
				userErr = errors.New("user client not configured in thread handler for hydration")
				return
			}
			resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: authorIDs})
			if err != nil { userErr = err; return }
			authorsMap = resp.GetUsers()
		}()
	}
	if len(mediaIDs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if h.mediaClient == nil {
				log.Println("GetUserSpecificThreads: mediaClient in ThreadHandler is nil")
				mediaErr = errors.New("media client not configured in thread handler for hydration")
				return
			}
			resp, err := h.mediaClient.GetMultipleMediaMetadata(c.Request.Context(), &mediapb.GetMultipleMediaMetadataRequest{MediaIds: mediaIDs})
			if err != nil { mediaErr = err; return }
			mediaMap = resp.GetMediaItems()
		}()
	}
	wg.Wait()
    if userErr != nil { log.Printf("Error fetching authors for user threads: %v", userErr)}
    if mediaErr != nil { log.Printf("Error fetching media for user threads: %v", mediaErr)}


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

func (h *ThreadHandler) GetBookmarkedThreadsHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}
	if requesterUserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required to view bookmarks"})
		return
	}

	page, limit := parsePagination(c)

	grpcReq := &threadpb.GetBookmarkedThreadsRequest{
		UserId:         requesterUserID,
		RequesterUserId: &requesterUserID,
		Page:           page,
		Limit:          limit,
	}

	threadServiceResp, err := h.threadClient.GetBookmarkedThreads(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get bookmarked threads", err)
		return
	}

	// --- Hydrate Author and Media for the fetched bookmarked threads ---
    if len(threadServiceResp.GetThreads()) == 0 {
        c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false})
        return
    }
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
	for id := range authorIDsSet {
		authorIDs = append(authorIDs, id)
	}
	var mediaIDs []uint32
	for id := range mediaIDsSet {
		mediaIDs = append(mediaIDs, id)
	}
	var wg sync.WaitGroup
	var authorsMap map[uint32]*userpb.User
	var mediaMap map[uint32]*mediapb.Media
	var userErr, mediaErr error
    if len(authorIDs) > 0 {
        wg.Add(1); go func() { defer wg.Done()
            resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: authorIDs})
            if err != nil { userErr = err; return }; authorsMap = resp.GetUsers()
        }()
    }
    if len(mediaIDs) > 0 {
        wg.Add(1); go func() { defer wg.Done()
            resp, err := h.mediaClient.GetMultipleMediaMetadata(c.Request.Context(), &mediapb.GetMultipleMediaMetadataRequest{MediaIds: mediaIDs})
            if err != nil { mediaErr = err; return }; mediaMap = resp.GetMediaItems()
        }()
    }
    wg.Wait()
    if userErr != nil { log.Printf("Error fetching authors for bookmarks: %v", userErr) }
    if mediaErr != nil { log.Printf("Error fetching media for bookmarks: %v", mediaErr) }
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
// Helper to get combined list of users to exclude (blocked by me + blocking me)
func (h *ThreadHandler) getFeedExclusionIDs(ctx context.Context, requesterID uint32) ([]uint32, error) {
	if requesterID == 0 { return nil, nil } // No exclusions for unauthenticated

	var excludeIDs []uint32
	var wg sync.WaitGroup
	var errBlocked, errBlocking error
	var blockedByRequester, requesterBlockedBy *userpb.UserIDListResponse

	wg.Add(2)
	go func() {
		defer wg.Done()
		blockedByRequester, errBlocked = h.userClient.GetBlockedUserIDs(ctx, &userpb.SocialListRequest{UserId: requesterID, Limit: 10000})
	}()
	go func() {
		defer wg.Done()
		requesterBlockedBy, errBlocking = h.userClient.GetBlockingUserIDs(ctx, &userpb.SocialListRequest{UserId: requesterID, Limit: 10000})
	}()
	wg.Wait()

	if errBlocked != nil { return nil, fmt.Errorf("failed to get users blocked by requester: %w", errBlocked) }
	if errBlocking != nil { return nil, fmt.Errorf("failed to get users blocking requester: %w", errBlocking) }

	tempSet := make(map[uint32]bool)
	if blockedByRequester != nil {
		for _, id := range blockedByRequester.GetUserIds() { tempSet[id] = true }
	}
	if requesterBlockedBy != nil {
		for _, id := range requesterBlockedBy.GetUserIds() { tempSet[id] = true }
	}
	for id := range tempSet { excludeIDs = append(excludeIDs, id) }
	return excludeIDs, nil
}

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
		BookmarkCount:               tProto.GetBookmarkCount(),
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
				Email: authorProto.GetEmail(), ProfilePicture: authorProto.GetProfilePicture(), AccountPrivacy: authorProto.GetAccountPrivacy(),
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