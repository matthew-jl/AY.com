package http

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	threadpb "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/genproto/proto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ThreadHandler struct {
	threadClient *client.ThreadClient
	mediaClient  *client.MediaClient // Inject media client for potential uploads before thread creation
}

func NewThreadHandler(threadClient *client.ThreadClient, mediaClient *client.MediaClient) *ThreadHandler {
	return &ThreadHandler{threadClient: threadClient, mediaClient: mediaClient}
}

// Payload for creating a thread (matches frontend structure)
type CreateThreadPayload struct {
	Content          string   `form:"content"`                     // Use form binding if potentially mixed with files
	ParentThreadID   *uint32  `form:"parent_thread_id"`          // Optional
	ReplyRestriction string   `form:"reply_restriction"`         // e.g., "EVERYONE", "FOLLOWING"
	ScheduledAt      *string  `form:"scheduled_at"`              // Optional ISO 8601 string
	CommunityID      *uint32  `form:"community_id"`              // Optional
	MediaIDs         []uint32 `form:"media_ids,omitempty"`       // Optional: IDs from previous uploads
	// Note: Files would be handled via c.FormFile("media_files")
}


// CreateThread handles creating a new thread, possibly with media uploads first.
// For simplicity NOW, assumes media_ids are provided if media exists.
// A more robust flow would handle file uploads within this request or via separate /media/upload calls first.
func (h *ThreadHandler) CreateThread(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok { return /* Error handled in helper */ }

	var payload CreateThreadPayload
	// Use ShouldBind for flexibility (handles JSON, form data)
	if err := c.ShouldBind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
		return
	}

	// Basic validation
	if payload.Content == "" && len(payload.MediaIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thread must contain content or media"})
		return
	}

	// Prepare gRPC request
	grpcReq := &threadpb.CreateThreadRequest{
		UserId:   userID,
		Content:  payload.Content,
		MediaIds: payload.MediaIDs, // Directly use provided IDs
	}

	// Map Reply Restriction string to enum
	grpcReq.ReplyRestriction = mapHTTPReplyRestrictionToProto(payload.ReplyRestriction)

	// Handle optional fields
	if payload.ParentThreadID != nil {
		grpcReq.ParentThreadId = payload.ParentThreadID
	}
	if payload.CommunityID != nil {
		grpcReq.CommunityId = payload.CommunityID
	}
	if payload.ScheduledAt != nil && *payload.ScheduledAt != "" {
		// Parse timestamp string (e.g., ISO 8601)
		t, err := time.Parse(time.RFC3339, *payload.ScheduledAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled_at format. Use ISO 8601 (RFC3339)."})
			return
		}
		grpcReq.ScheduledAt = timestamppb.New(t)
	}

	// --- TODO: Handle direct file uploads if needed ---
	// If files are uploaded with this request (multipart/form-data):
	// 1. Get files using c.FormFile or c.MultipartForm
	// 2. Loop through files, call h.mediaClient.UploadMedia for each
	// 3. Collect the returned media IDs and ADD them to grpcReq.MediaIds

	// Call Thread Service
	createdThread, err := h.threadClient.CreateThread(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "create thread", err)
		return
	}

	c.JSON(http.StatusCreated, createdThread) // Return the created thread
}

// GetThread retrieves a single thread by ID.
func (h *ThreadHandler) GetThread(c *gin.Context) {
	threadID, ok := getUint32Param(c, "threadId")
	if !ok { return }

	grpcReq := &threadpb.GetThreadRequest{ThreadId: threadID}
	thread, err := h.threadClient.GetThread(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get thread", err)
		return
	}
	c.JSON(http.StatusOK, thread)
}

// DeleteThread deletes a thread if the user is the owner.
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

// handleInteraction is a helper for like/unlike/bookmark/unbookmark
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
		// Allow "already exists" or "not found" errors for idempotent actions
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.AlreadyExists || st.Code() == codes.NotFound) {
            // Consider these successful for idempotency
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
	userID, _ := getUserIDFromContext(c) // Get user ID, ignore error if not logged in (for public feed?) - adjust as needed

	// Get query parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	feedType := c.DefaultQuery("type", "foryou") // Default to 'foryou'

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 { page = 1 }
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 { limit = 20 }


	grpcReq := &threadpb.GetFeedThreadsRequest{
		UserId:   userID, // Pass user ID for potential 'following' logic
		Page:     int32(page),
		Limit:    int32(limit),
		FeedType: feedType,
	}

	resp, err := h.threadClient.GetFeedThreads(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get feed threads", err)
		return
	}

	// TODO: Hydrate threads with Author and Media data here
	// 1. Collect all user IDs and media IDs from resp.Threads
	// 2. Make batch calls to User Service (GetUserProfilesByIds - needs implementation)
	// 3. Make batch calls to Media Service (GetMediaMetadataByIds - needs implementation)
	// 4. Map the results back onto the threads before sending response

	c.JSON(http.StatusOK, resp) // Send GetFeedThreadsResponse (includes threads and hasMore)
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

// Centralized gRPC error handling for HTTP responses
func handleGRPCError(c *gin.Context, operation string, err error) {
	st, ok := status.FromError(err)
	log.Printf("gRPC error during '%s': Code=%s, Msg=%s", operation, st.Code(), st.Message()) // Log details
	if ok {
		httpCode := grpcStatusCodeToHTTP(st.Code()) // Use existing helper
		c.JSON(httpCode, gin.H{"error": st.Message()})
	} else {
		// Non-gRPC error
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to %s: %s", operation, err.Error())})
	}
}