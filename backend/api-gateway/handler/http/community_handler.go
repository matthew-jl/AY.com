// backend/api-gateway/handler/http/community_handler.go
package http

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	communitypb "github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/genproto/proto"
	"github.com/gin-gonic/gin"
	// "google.golang.org/grpc/status" // For handleGRPCError
	// "google.golang.org/grpc/codes"  // For handleGRPCError
)

type CommunityHandler struct {
	communityClient *client.CommunityClient
	userClient   *client.UserClient
	// mediaClient  *client.MediaClient  // If gateway handles media URL validation/proxying
}

// NewCommunityHandler initializes a new community handler
func NewCommunityHandler(cc *client.CommunityClient , uc *client.UserClient/* , mc *client.MediaClient */) *CommunityHandler {
	return &CommunityHandler{
		communityClient: cc,
		userClient:   uc,
		// mediaClient:  mc,
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


// CommunityServiceHealthHTTP
func (h *CommunityHandler) CommunityServiceHealthHTTP(c *gin.Context) {
    resp, err := h.communityClient.HealthCheck(c.Request.Context())
    if err != nil {handleGRPCError(c, "community service health", err); return }
    c.JSON(http.StatusOK, resp)
}