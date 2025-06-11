package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	communitypb "github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/genproto/proto"
	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	threadpb "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/genproto/proto"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/gin-gonic/gin"
	// "google.golang.org/grpc/status" // For handleGRPCError
	// "google.golang.org/grpc/codes"  // For handleGRPCError
)

type CommunityHandler struct {
	communityClient *client.CommunityClient
	userClient   *client.UserClient
	threadClient *client.ThreadClient
	mediaClient  *client.MediaClient
}

// NewCommunityHandler initializes a new community handler
func NewCommunityHandler(cc *client.CommunityClient , uc *client.UserClient, tc *client.ThreadClient, mc *client.MediaClient) *CommunityHandler {
	return &CommunityHandler{
		communityClient: cc,
		userClient:   uc,
		threadClient: tc,
		mediaClient:  mc,
	}
}

// Payload for creating a community from HTTP request
type CreateCommunityPayloadHTTP struct {
	Name        string   `json:"name" binding:"required,min=3,max=100"`
	Description string   `json:"description" binding:"max=1000"`
	IconURL     string   `json:"icon_url" binding:"omitempty,url"`   // Optional, must be a valid URL if provided
	BannerURL   string   `json:"banner_url" binding:"omitempty,url"` // Optional
	Categories  []string `json:"categories"`                         // Can be empty
	Rules       []string `json:"rules"`                              // Can be empty
}

type HandleJoinRequestPayloadHTTP struct {
	// For admin/mod to accept/reject a specific user's request for their community
	TargetUserID uint32 `json:"target_user_id" binding:"required"`
}

type UpdateMemberRolePayloadHTTP struct {
	TargetUserID uint32 `json:"target_user_id" binding:"required"`
	NewRole      string `json:"new_role" binding:"required,oneof=member moderator"` // Validate role
}

// CreateCommunityHTTP handles the HTTP request to create a new community.
func (h *CommunityHandler) CreateCommunityHTTP(c *gin.Context) {
	creatorID, ok := getUserIDFromContext(c) // Uses your existing helper
	if !ok {
		return
	}
	if creatorID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required to create a community"})
		return
	}

	var payload CreateCommunityPayloadHTTP
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("CreateCommunityHTTP: Invalid request payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
		return
	}

	// TODO: Add more specific business logic validation if needed here
	// e.g., check profanity in name/description, category validation against a predefined list.

	grpcReq := &communitypb.CreateCommunityRequest{
		CreatorId:   creatorID,
		Name:        payload.Name,
		Description: payload.Description,
		IconUrl:     payload.IconURL,   // Pass URL from frontend
		BannerUrl:   payload.BannerURL, // Pass URL from frontend
		Categories:  payload.Categories,
		Rules:       payload.Rules,
	}

	createdCommunityDetails, err := h.communityClient.CreateCommunity(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "create community", err) // Use your existing helper
		return
	}

	// The response 'createdCommunityDetails' is already *communitypb.CommunityDetails
	// Gin will marshal this proto message to JSON.
	c.JSON(http.StatusCreated, createdCommunityDetails)
}

// GetCommunityDetailsHTTP retrieves details for a specific community by ID only.
func (h *CommunityHandler) GetCommunityDetailsHTTP(c *gin.Context) {
	communityID, ok := getUint32Param(c, "communityId")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing community ID"})
		return
	}
	requesterUserID, _ := getUserIDFromContext(c)

	grpcReq := &communitypb.GetCommunityDetailsRequest{
		CommunityId:     communityID,
		RequesterUserId: &requesterUserID,
	}
	resp, err := h.communityClient.GetCommunityDetails(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get community details", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListCommunitiesHTTP lists communities based on filters.
func (h *CommunityHandler) ListCommunitiesHTTP(c *gin.Context) {
	page, limit := parsePagination(c)
	filterTypeStr := c.DefaultQuery("filter_type", "ALL_PUBLIC")
	userIDContextStr := c.Query("user_id_context") // For JOINED_BY_USER, CREATED_BY_USER
	searchQuery := c.Query("search_query")
	categoriesStr := c.Query("categories")

	var userIDCtx uint32
	if userIDContextStr != "" { uid, _ := strconv.ParseUint(userIDContextStr, 10, 32); userIDCtx = uint32(uid) }
	requesterUserID, _ := getUserIDFromContext(c)
	var categories []string; if categoriesStr != "" { categories = strings.Split(categoriesStr, ",") }

	filterTypeVal, ok := communitypb.ListCommunitiesRequest_FilterType_value[strings.ToUpper(filterTypeStr)]
	if !ok { filterTypeVal = int32(communitypb.ListCommunitiesRequest_ALL_PUBLIC) }

	grpcReq := &communitypb.ListCommunitiesRequest{
		FilterType:      communitypb.ListCommunitiesRequest_FilterType(filterTypeVal),
		UserIdContext:    &userIDCtx,
		RequesterUserId:  &requesterUserID,
		Page:            page, Limit:           limit,
		SearchQuery:     searchQuery, CategoryFilters: categories,
	}
	resp, err := h.communityClient.ListCommunities(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "list communities", err); return }
	// TODO: Hydrate creator summaries for each community in resp.Communities if not done fully by Community Service
	c.JSON(http.StatusOK, resp)
}

// RequestToJoinCommunityHTTP allows a user to request to join a community.
func (h *CommunityHandler) RequestToJoinCommunityHTTP(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	communityID, ok := getUint32Param(c, "communityId")
	if !ok { return }

	grpcReq := &communitypb.CommunityUserRequest{CommunityId: communityID, UserId: requesterUserID}
	_, err := h.communityClient.RequestToJoinCommunity(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "request to join community", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Join request sent successfully"})
}

// AcceptJoinRequestHTTP allows a mod/owner to accept a join request.
func (h *CommunityHandler) AcceptJoinRequestHTTP(c *gin.Context) {
	actorUserID, ok := getUserIDFromContext(c) // User performing the action
	if !ok { return }
	communityID, ok := getUint32Param(c, "communityId")
	if !ok { return }
	// requestID, ok := getUint32Param(c, "requestId") // If operating on request ID
	// if !ok { return }

	var payload HandleJoinRequestPayloadHTTP // Expects target_user_id in body
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()}); return
	}


	grpcReq := &communitypb.CommunityUserActionRequest{
		CommunityId:   communityID,
		TargetUserId:  payload.TargetUserID, // User whose request is being accepted
		ActorUserId:   actorUserID,
	}
	_, err := h.communityClient.AcceptJoinRequest(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "accept join request", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Join request accepted"})
}

// RejectJoinRequestHTTP allows a mod/owner to reject a join request.
func (h *CommunityHandler) RejectJoinRequestHTTP(c *gin.Context) {
	actorUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	communityID, ok := getUint32Param(c, "communityId")
	if !ok { return }
	// requestID, ok := getUint32Param(c, "requestId")
	// if !ok { return }
	var payload HandleJoinRequestPayloadHTTP
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()}); return
	}

	grpcReq := &communitypb.CommunityUserActionRequest{
		CommunityId:   communityID,
		TargetUserId:  payload.TargetUserID,
		ActorUserId:   actorUserID,
	}
	_, err := h.communityClient.RejectJoinRequest(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "reject join request", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Join request rejected"})
}

// GetCommunityMembersHTTP retrieves members of a community.
func (h *CommunityHandler) GetCommunityMembersHTTP(c *gin.Context) {
	communityID, ok := getUint32Param(c, "communityId")
	if !ok { return }
	requesterUserID, _ := getUserIDFromContext(c) // Optional for checking follow status
	page, limit := parsePagination(c)
	roleFilter := c.DefaultQuery("role", "all")

	grpcReq := &communitypb.GetCommunityMembersRequest{
		CommunityId:    communityID,
		RequesterUserId: &requesterUserID,
		RoleFilter:     roleFilter,
		Page:           page, Limit:          limit,
	}
	resp, err := h.communityClient.GetCommunityMembers(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "get community members", err); return }
	// Community Service should hydrate UserSummary for members.
	c.JSON(http.StatusOK, resp)
}

// GetUserJoinRequestsHTTP retrieves join requests made by the authenticated user.
func (h *CommunityHandler) GetUserJoinRequestsHTTP(c *gin.Context) {
    requesterUserID, ok := getUserIDFromContext(c)
    if !ok { return }
    page, limit := parsePagination(c)

    grpcReq := &communitypb.GetUserJoinRequestsRequest{UserId: requesterUserID, Page: page, Limit: limit}
    resp, err := h.communityClient.GetUserJoinRequests(c.Request.Context(), grpcReq)
    if err != nil { handleGRPCError(c, "get user join requests", err); return }
    // Community Service should hydrate details within JoinRequestDetails
    c.JSON(http.StatusOK, resp)
}

// GetCommunityPendingRequestsHTTP retrieves pending join requests for a community (for mods/admins).
func (h *CommunityHandler) GetCommunityPendingRequestsHTTP(c *gin.Context) {
    actorUserID, ok := getUserIDFromContext(c)
    if !ok { return }
    communityID, ok := getUint32Param(c, "communityId")
    if !ok { return }
    page, limit := parsePagination(c)

    grpcReq := &communitypb.GetCommunityPendingRequestsRequest{
        CommunityId: communityID, ActorUserId: actorUserID, Page: page, Limit: limit,
    }
    resp, err := h.communityClient.GetCommunityPendingRequests(c.Request.Context(), grpcReq)
    if err != nil { handleGRPCError(c, "get community pending requests", err); return }
    c.JSON(http.StatusOK, resp)
}

func (h *CommunityHandler) UpdateMemberRoleHTTP(c *gin.Context) {
	actorUserID, ok := getUserIDFromContext(c)
	if !ok { return }
	communityID, ok := getUint32Param(c, "communityId")
	if !ok { return }

	var payload UpdateMemberRolePayloadHTTP
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()}); return
	}

	grpcReq := &communitypb.UpdateMemberRoleRequest{
		CommunityId:   communityID,
		ActorUserId:   actorUserID,
		TargetUserId:  payload.TargetUserID,
		NewRole:       payload.NewRole,
	}
	_, err := h.communityClient.UpdateMemberRole(c.Request.Context(), grpcReq)
	if err != nil { handleGRPCError(c, "update member role", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Member role updated successfully"})
}

func (h *CommunityHandler) GetCommunityThreadsHTTP(c *gin.Context) {
	communityID, ok := getUint32Param(c, "communityId")
	if !ok { return }
	if communityID == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "Community ID is required"}); return }

	requesterUserID, _ := getUserIDFromContext(c)
	page, limit := parsePagination(c)
	sortType := c.DefaultQuery("sort", "latest") // e.g., "latest", "top"

	// --- Permission Check: Can requester view this community's content? ---
    commDetailsResp, err := h.communityClient.GetCommunityDetails(c.Request.Context(), &communitypb.GetCommunityDetailsRequest{
        CommunityId:     uint32(communityID),
        RequesterUserId: &requesterUserID,
    })
    if err != nil { handleGRPCError(c, "check community visibility for threads", err); return }
    if commDetailsResp == nil || commDetailsResp.Community == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Community not found"}); return
    }
    // Apply privacy rules from community details before fetching threads
    if commDetailsResp.Community.Status != communitypb.CommunityStatus_ACTIVE {
         c.JSON(http.StatusForbidden, gin.H{"error": "Community is not active"}); return
    }

    // Get IDs to exclude based on requester's blocks
    excludeUserIDs, err := h.getFeedExclusionIDs(c.Request.Context(), requesterUserID) // Use the existing helper
    if err != nil {
        log.Printf("GetCommunityThreadsHTTP: Error getting exclusion IDs: %v", err)
        // Potentially proceed without exclusions or return an error
    }


	// Call Thread Service's GetCommunityThreads RPC
	grpcReq := &threadpb.GetCommunityThreadsRequest{
		CommunityId:    uint32(communityID), // Cast to uint32
		RequesterUserId: &requesterUserID,    // For is_liked etc. on threads
		SortType:       sortType,
		Page:           page,
		Limit:          limit,
        ExcludeUserIds: excludeUserIDs,
	}

	threadServiceResp, err := h.threadClient.GetCommunityThreads(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get community threads from thread service", err)
		return
	}

	// --- Hydrate Author and Media (same logic as GetFeed) ---
    if len(threadServiceResp.GetThreads()) == 0 {
        c.JSON(http.StatusOK, FrontendFeedResponse{Threads: []FrontendThreadData{}, HasMore: false})
        return
    }
    authorIDsSet := make(map[uint32]bool); mediaIDsSet := make(map[uint32]bool)
    for _, t := range threadServiceResp.GetThreads() {
        if t.GetUserId() != 0 { authorIDsSet[t.GetUserId()] = true }
        for _, mediaID := range t.GetMediaIds() { if mediaID != 0 { mediaIDsSet[mediaID] = true } }
    }
    var authorIDs []uint32; for id := range authorIDsSet { authorIDs = append(authorIDs, id) }
    var mediaIDs []uint32; for id := range mediaIDsSet { mediaIDs = append(mediaIDs, id) }
    var wg sync.WaitGroup; var authorsMap map[uint32]*userpb.User; var mediaMap map[uint32]*mediapb.Media
    var userErr, mediaErr error
    if len(authorIDs) > 0 && h.userClient != nil {
        wg.Add(1); go func() { defer wg.Done();
            resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: authorIDs})
            if err != nil { userErr = err; return }; authorsMap = resp.GetUsers()
        }()
    }
    if len(mediaIDs) > 0 && h.mediaClient != nil {
        wg.Add(1); go func() { defer wg.Done();
            resp, err := h.mediaClient.GetMultipleMediaMetadata(c.Request.Context(), &mediapb.GetMultipleMediaMetadataRequest{MediaIds: mediaIDs})
            if err != nil { mediaErr = err; return }; mediaMap = resp.GetMediaItems()
        }()
    }
    wg.Wait()
    if userErr != nil { log.Printf("Error fetching authors for community threads: %v", userErr) }
    if mediaErr != nil { log.Printf("Error fetching media for community threads: %v", mediaErr) }
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

func (h *CommunityHandler) GetTopCommunityMembersHTTP(c *gin.Context) {
	communityID, ok := getUint32Param(c, "communityId") // Expecting ID
	if !ok { return }

	requesterUserID, _ := getUserIDFromContext(c) // For context if needed by GetUserProfile

	limitStr := c.DefaultQuery("limit", "3")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 5 { limit = 3 }

	// 1. Fetch a list of member UserSummaries from Community Service
	communityMembersResp, err := h.communityClient.GetCommunityMembers(c.Request.Context(), &communitypb.GetCommunityMembersRequest{
		CommunityId:    communityID,
		RequesterUserId: &requesterUserID, // For any context needed by GetCommunityMembers
		RoleFilter:     "all",           // Get all roles
		Page:           1,
		Limit:          50,
	})
	if err != nil {
		handleGRPCError(c, "get community members for top list", err)
		return
	}
	if communityMembersResp == nil || len(communityMembersResp.GetMembers()) == 0 {
		c.JSON(http.StatusOK, gin.H{"users": []FrontendUserProfile{}}) // Return empty list
		return
	}

	// 2. Collect User IDs of these members to fetch their full profiles (with follower counts)
	memberUserIDs := make([]uint32, 0, len(communityMembersResp.GetMembers()))
	memberUserSummariesMap := make(map[uint32]*communitypb.UserSummary) // To keep basic info if full profile fetch fails

	for _, memberDetail := range communityMembersResp.GetMembers() {
		if memberDetail.GetUser() != nil {
			memberUserIDs = append(memberUserIDs, memberDetail.GetUser().GetId())
            memberUserSummariesMap[memberDetail.GetUser().GetId()] = memberDetail.GetUser()
		}
	}

	if len(memberUserIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"users": []FrontendUserProfile{}})
		return
	}

	// 3. Fetch full profiles (which include follower_count) from User Service for these member IDs
	// We need UserProfileResponse from user.proto which includes follower_count
	fullMemberProfilesWithStats := make([]*userpb.UserProfileResponse, 0, len(memberUserIDs))

    // Fetch profiles one by one (N+1, bad for many members)
    if h.userClient != nil {
        var profilesWg sync.WaitGroup
        profilesChan := make(chan *userpb.UserProfileResponse, len(memberUserIDs))

        for _, memberID := range memberUserIDs {
            profilesWg.Add(1)
            go func(uid uint32) {
                defer profilesWg.Done()
                profileResp, err := h.userClient.GetUserProfile(c.Request.Context(), &userpb.GetUserProfileRequest{
                    UserIdToView:    uid,
                    RequesterUserId: &requesterUserID, // Pass requester for context
                })
                if err != nil {
                    log.Printf("Error fetching profile for member %d (for top list): %v", uid, err)
                    // fetchProfileErr = err // Store first error, or collect all
                    return
                }
                if profileResp != nil {
                    profilesChan <- profileResp
                }
            }(memberID)
        }

        go func() {
            profilesWg.Wait()
            close(profilesChan)
        }()

        for profileResp := range profilesChan {
            fullMemberProfilesWithStats = append(fullMemberProfilesWithStats, profileResp)
        }
    }
    // Could check fetchProfileErr here if it's critical


	// 4. Sort these full profiles by follower_count (descending)
	sort.SliceStable(fullMemberProfilesWithStats, func(i, j int) bool {
		return fullMemberProfilesWithStats[i].GetFollowerCount() > fullMemberProfilesWithStats[j].GetFollowerCount()
	})

	// 5. Take the top N (up to the requested limit)
	finalCount := limit
	if len(fullMemberProfilesWithStats) < limit {
		finalCount = len(fullMemberProfilesWithStats)
	}
	topSortedProfiles := fullMemberProfilesWithStats[:finalCount]

	// 6. Map to FrontendUserProfile structure
	frontendTopUsers := make([]FrontendUserProfile, 0, len(topSortedProfiles))
	for _, profileWithStats := range topSortedProfiles {
		if profileWithStats.GetUser() != nil {
			pbUser := profileWithStats.GetUser()
			frontendTopUsers = append(frontendTopUsers, FrontendUserProfile{
				ID:             pbUser.GetId(),
				Name:           pbUser.GetName(),
				Username:       pbUser.GetUsername(),
				ProfilePicture: pbUser.GetProfilePicture(),
                Email:          pbUser.GetEmail(),
                AccountPrivacy: pbUser.GetAccountPrivacy(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"users": frontendTopUsers})
}

func (h *CommunityHandler) CommunityServiceHealthHTTP(c *gin.Context) {
    resp, err := h.communityClient.HealthCheck(c.Request.Context())
    if err != nil {handleGRPCError(c, "community service health", err); return }
    c.JSON(http.StatusOK, resp)
}

// Helpers
func (h *CommunityHandler) getFeedExclusionIDs(ctx context.Context, requesterID uint32) ([]uint32, error) {
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