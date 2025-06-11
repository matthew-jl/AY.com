package grpc

import (
	"context"
	"fmt"
	"log"
	"strings"

	communitypb "github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/community-service/repository/postgres"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"

	// "github.com/lib/pq" // Not needed directly in handler for pq.StringArray with GORM
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CommunityHandler struct {
	communitypb.UnimplementedCommunityServiceServer
	repo *postgres.CommunityRepository
	userClient  userpb.UserServiceClient   // for hydration
}

// Inject repository (and potentially other clients later)
func NewCommunityHandler(repo *postgres.CommunityRepository, uc userpb.UserServiceClient) *CommunityHandler {
	return &CommunityHandler{repo: repo, userClient: uc}
}

func (h *CommunityHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*communitypb.HealthResponse, error) {
	log.Println("Community Service Health Check received")
    if err := h.repo.CheckHealth(ctx); err != nil {
        log.Printf("Community Service health check failed: %v", err)
        return &communitypb.HealthResponse{Status: "Community Service DEGRADED"}, nil
    }
	return &communitypb.HealthResponse{Status: "Community Service OK"}, nil
}

func (h *CommunityHandler) CreateCommunity(ctx context.Context, req *communitypb.CreateCommunityRequest) (*communitypb.CommunityDetails, error) {
	log.Printf("CreateCommunity request by User %d, Name: %s", req.CreatorId, req.Name)
	if req.CreatorId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Creator ID is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Community name cannot be empty")
	}
	if len(req.Name) > 100 {
		return nil, status.Errorf(codes.InvalidArgument, "Community name too long (max 100 chars)")
	}
	if len(req.Description) > 1000 { // Example limit
        return nil, status.Errorf(codes.InvalidArgument, "Description too long (max 1000 chars)")
    }
    // TODO: Validate categories, rules (e.g., max number, length of each)
    // TODO: Validate icon_url and banner_url format (are they valid URLs?) - or Media Service does this

	dbCommunity := &postgres.Community{
		Name:        req.Name,
		Description: req.Description,
		CreatorID:   uint(req.CreatorId),
		IconURL:     req.IconUrl,
		BannerURL:   req.BannerUrl,
		Categories:  req.Categories, // pq.StringArray handles []string directly
		Rules:       req.Rules,
		// Status defaults to "pending_approval" in DB model
	}

	err := h.repo.CreateCommunity(ctx, dbCommunity)
	if err != nil {
		log.Printf("Failed to create community '%s': %v", req.Name, err)
		if err.Error() == "community name already exists" {
			return nil, status.Errorf(codes.AlreadyExists, "A community with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "Could not create community")
	}

	log.Printf("Community '%s' (ID: %d) created successfully, pending approval.", dbCommunity.Name, dbCommunity.ID)

	// Map DB model to proto response
	var creatorSummary *communitypb.UserSummary
	if dbCommunity.CreatorID != 0 && h.userClient != nil {
		creatorSummaries, err := h.hydrateUserSummaries(ctx, []uint32{uint32(dbCommunity.CreatorID)})
		if err == nil && creatorSummaries != nil {
			creatorSummary = creatorSummaries[uint32(dbCommunity.CreatorID)]
		} else { log.Printf("Error hydrating creator %d for community %d: %v", dbCommunity.CreatorID, dbCommunity.ID, err)}
	}

	// For now, member count is 1 (the creator)
	return mapDBCommunityToProtoDetails(dbCommunity, creatorSummary, 1), nil
}

func (h *CommunityHandler) GetCommunityDetails(ctx context.Context, req *communitypb.GetCommunityDetailsRequest) (*communitypb.CommunityDetailsResponse, error) {
	log.Printf("GetCommunityDetails request for ID: %d, Requester: %d", req.CommunityId, req.GetRequesterUserId())
	if req.CommunityId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Community ID required") }

	dbCommunity, err := h.repo.GetCommunityByID(ctx, uint(req.CommunityId))
	if err != nil { /* ... handle not found ... */ }

    // Hydrate creator summary
    var creatorSummary *communitypb.UserSummary
    if dbCommunity.CreatorID != 0 && h.userClient != nil {
        creatorSummaries, err := h.hydrateUserSummaries(ctx, []uint32{uint32(dbCommunity.CreatorID)})
        if err == nil && creatorSummaries != nil {
            creatorSummary = creatorSummaries[uint32(dbCommunity.CreatorID)]
        } else { log.Printf("Error hydrating creator %d for community %d: %v", dbCommunity.CreatorID, dbCommunity.ID, err)}
    }

    memberCount, _ := h.repo.CountCommunityMembers(ctx, dbCommunity.ID) // Get member count
	detailsProto := mapDBCommunityToProtoDetails(dbCommunity, creatorSummary, memberCount)

    response := &communitypb.CommunityDetailsResponse{Community: detailsProto}

    // Populate requester-specific fields
    if req.GetRequesterUserId() != 0 {
        role, errRole := h.repo.GetUserRoleInCommunity(ctx, dbCommunity.ID, uint(req.GetRequesterUserId()))
        if errRole != nil {
            log.Printf("Error getting user role in GetCommunityDetails for comm %d, user %d: %v", dbCommunity.ID, req.GetRequesterUserId(), errRole)
        }
        response.RequesterRole = role
        if role == "pending_join" {
            response.HasPendingRequestByRequester = true
        }
        if role == "member" || role == "moderator" || role == "owner" {
             response.IsJoinedByRequester = true
        }
    }

	return response, nil
}


func (h *CommunityHandler) ListCommunities(ctx context.Context, req *communitypb.ListCommunitiesRequest) (*communitypb.ListCommunitiesResponse, error) {
	log.Printf("ListCommunities request: Filter=%s, UserContext=%d, RequesterID: %d", req.FilterType, req.GetUserIdContext(), req.GetRequesterUserId())
	limit, offset := getLimitOffsetCommunity(req.Page, req.Limit)

	params := postgres.ListCommunitiesParams{
		Limit:           limit,
		Offset:          offset,
		FilterType:      req.FilterType.String(),
		CategoryFilters: req.CategoryFilters,
	}
	if req.GetUserIdContext() != 0 { uid := uint(req.GetUserIdContext()); params.UserIDContext = &uid }
	if req.SearchQuery != "" { params.SearchQuery = &req.SearchQuery }

	dbCommunities, err := h.repo.ListCommunities(ctx, params)
	if err != nil {
        log.Printf("Error listing communities from repo: %v", err)
        return nil, status.Errorf(codes.Internal, "Failed to list communities")
    }

	protoCommunities := make([]*communitypb.Community, 0, len(dbCommunities))
	if len(dbCommunities) > 0 {
		communityIDs := make([]uint, len(dbCommunities))
		for i, c := range dbCommunities { communityIDs[i] = c.ID }

		// Batch fetch member counts
		memberCountsMap, errCounts := h.repo.GetMemberCountsForMultipleCommunities(ctx, communityIDs) // NEW REPO METHOD
		if errCounts != nil { log.Printf("ListCommunities: Error fetching member counts: %v", errCounts) }

		// Batch fetch join status and pending request status for the requester
		requesterJoinStatusMap := make(map[uint]string) // map[communityID]roleOrPending
		if req.GetRequesterUserId() != 0 {
            // NEW REPO METHOD: GetUserRolesAndPendingRequestsForCommunities
			requesterJoinStatusMap, err = h.repo.GetUserRolesAndPendingRequestsForCommunities(ctx, uint(req.GetRequesterUserId()), communityIDs)
			if err != nil {
				log.Printf("ListCommunities: Error fetching requester join statuses: %v", err)
			}
		}

		for _, c := range dbCommunities {
			memberCount := int64(0)
			if counts, ok := memberCountsMap[c.ID]; ok {
				memberCount = counts
			}

            isJoined := false
            hasPending := false
            if req.GetRequesterUserId() != 0 {
                roleOrStatus := requesterJoinStatusMap[c.ID] // This will be role like "member", "owner", or "pending_join", or ""
                if roleOrStatus == "member" || roleOrStatus == "moderator" || roleOrStatus == "owner" {
                    isJoined = true
                }
                if roleOrStatus == "pending_join" {
                    hasPending = true
                }
            }

			protoCommunities = append(protoCommunities, mapDBCommunityToListProto(&c, memberCount, isJoined, hasPending))
		}
	}
	return &communitypb.ListCommunitiesResponse{Communities: protoCommunities, HasMore: len(dbCommunities) == limit}, nil
}


func (h *CommunityHandler) RequestToJoinCommunity(ctx context.Context, req *communitypb.CommunityUserRequest) (*emptypb.Empty, error) {
    log.Printf("RequestToJoinCommunity: User %d to Community %d", req.UserId, req.CommunityId)
    if req.CommunityId == 0 || req.UserId == 0 { /* ... invalid arg ...*/ }
    // TODO: Check community status (must be active) and if it's invite-only etc.
    // TODO: Check if user is already blocked from this community
    err := h.repo.CreateJoinRequest(ctx, uint(req.CommunityId), uint(req.UserId))
    if err != nil {
        if err.Error() == "already a member of this community" {
            return nil, status.Errorf(codes.FailedPrecondition, "%v", err.Error())
        }
        // "join request already pending" might be another error from CreateJoinRequest using OnConflict
        return nil, status.Errorf(codes.Internal, "Failed to create join request: %v", err)
    }
    // TODO: Notify community mods/owner via RabbitMQ
    return &emptypb.Empty{}, nil
}

func (h *CommunityHandler) AcceptJoinRequest(ctx context.Context, req *communitypb.CommunityUserActionRequest) (*emptypb.Empty, error) {
    log.Printf("AcceptJoinRequest: TargetUser %d for Community %d by Actor %d", req.TargetUserId, req.CommunityId, req.ActorUserId)
    if req.CommunityId == 0 || req.TargetUserId == 0 || req.ActorUserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "CommunityID, TargetUserID, and ActorUserID are required")
	}
    // TODO: Permission Check - Is ActorUserID a mod/owner of CommunityID?
    // Example: role, err := h.repo.GetUserRoleInCommunity(ctx, uint(req.CommunityId), uint(req.ActorUserId))
    // if err != nil || (role != "owner" && role != "moderator") {
    //  return nil, status.Errorf(codes.PermissionDenied, "Actor does not have permission to accept requests")
    // }

    // Find the PENDING join request for this user and community
    joinReq, err := h.repo.GetPendingJoinRequestByUserAndCommunity(ctx, uint(req.CommunityId), uint(req.TargetUserId))
    if err != nil {
        if err.Error() == "join request not found" {
            return nil, status.Errorf(codes.NotFound, "No pending join request found for this user in this community")
        }
        log.Printf("Error finding pending join request: %v", err)
        return nil, status.Errorf(codes.Internal, "Failed to find join request")
    }
    if joinReq == nil {
         return nil, status.Errorf(codes.NotFound, "No pending join request found")
    }


    err = h.repo.UpdateJoinRequestStatus(ctx, joinReq.ID, "accepted", uint(req.ActorUserId))
    if err != nil {
        log.Printf("Error accepting join request %d: %v", joinReq.ID, err)
        if err.Error() == "join request already resolved" {
             return nil, status.Error(codes.FailedPrecondition, err.Error())
        }
        return nil, status.Errorf(codes.Internal, "Failed to accept join request")
    }
    // TODO: Publish "join_request_accepted" and "new_community_member" events to RabbitMQ
    log.Printf("Join request for user %d to community %d accepted by user %d", req.TargetUserId, req.CommunityId, req.ActorUserId)
    return &emptypb.Empty{}, nil
}

func (h *CommunityHandler) RejectJoinRequest(ctx context.Context, req *communitypb.CommunityUserActionRequest) (*emptypb.Empty, error) {
    log.Printf("RejectJoinRequest: TargetUser %d for Community %d by Actor %d", req.TargetUserId, req.CommunityId, req.ActorUserId)
    // ... (similar validation and permission check as AcceptJoinRequest) ...

    joinReq, err := h.repo.GetPendingJoinRequestByUserAndCommunity(ctx, uint(req.CommunityId), uint(req.TargetUserId))
    if err != nil {
        if err.Error() == "join request not found" {
            return nil, status.Errorf(codes.NotFound, "No pending join request found for this user in this community")
        }
        log.Printf("Error finding pending join request: %v", err)
        return nil, status.Errorf(codes.Internal, "Failed to find join request")
    }
    if joinReq == nil {
         return nil, status.Errorf(codes.NotFound, "No pending join request found")
    }

    err = h.repo.UpdateJoinRequestStatus(ctx, joinReq.ID, "rejected", uint(req.ActorUserId))
    if err != nil {
        log.Printf("Error rejecting join request %d: %v", joinReq.ID, err)
        if err.Error() == "join request already resolved" {
             return nil, status.Error(codes.FailedPrecondition, err.Error())
        }
        return nil, status.Errorf(codes.Internal, "Failed to reject join request")
    }
    // TODO: Publish "join_request_rejected" event
    log.Printf("Join request for user %d to community %d rejected by user %d", req.TargetUserId, req.CommunityId, req.ActorUserId)
    return &emptypb.Empty{}, nil
}

func (h *CommunityHandler) GetCommunityMembers(ctx context.Context, req *communitypb.GetCommunityMembersRequest) (*communitypb.GetCommunityMembersResponse, error) {
	log.Printf("GetCommunityMembers for Community %d, RoleFilter: %s, Requester: %d", req.CommunityId, req.RoleFilter, req.GetRequesterUserId())
    if req.CommunityId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Community ID is required")
	}
    limit, offset := getLimitOffsetCommunity(req.Page, req.Limit)

    dbMembers, err := h.repo.GetCommunityMembers(ctx, uint(req.CommunityId), req.RoleFilter, limit, offset)
    if err != nil {
		log.Printf("GetCommunityMembers: Error fetching members for community %d: %v", req.CommunityId, err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch community members")
	}

    memberDetailsList := make([]*communitypb.CommunityMemberDetails, 0, len(dbMembers))
    if len(dbMembers) > 0 {
        userIDsToHydrate := make([]uint32, 0, len(dbMembers))
        for _, m := range dbMembers { userIDsToHydrate = append(userIDsToHydrate, uint32(m.UserID)) }

        userSummaries, err := h.hydrateUserSummaries(ctx, userIDsToHydrate)
        if err != nil { log.Printf("GetCommunityMembers: Error hydrating member summaries: %v", err) }

        for _, m := range dbMembers {
            detail := &communitypb.CommunityMemberDetails{
                Role:     m.Role,
                JoinedAt: timestamppb.New(m.JoinedAt),
            }
            if summary, ok := userSummaries[uint32(m.UserID)]; ok {
                detail.User = summary
            } else {
                detail.User = &communitypb.UserSummary{Id: uint32(m.UserID), Name: "Unknown User"} // Fallback
            }
            memberDetailsList = append(memberDetailsList, detail)
        }
    }
    return &communitypb.GetCommunityMembersResponse{Members: memberDetailsList, HasMore: len(dbMembers) == limit}, nil
}

func (h *CommunityHandler) GetUserJoinRequests(ctx context.Context, req *communitypb.GetUserJoinRequestsRequest) (*communitypb.GetUserJoinRequestsResponse, error) {
    log.Printf("GetUserJoinRequests for UserID: %d", req.UserId)
    if req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "User ID is required")}
    limit, offset := getLimitOffsetCommunity(req.Page, req.Limit)

    dbRequests, err := h.repo.GetUserJoinRequests(ctx, uint(req.UserId), limit, offset)
    if err != nil {
		log.Printf("GetUserJoinRequests: Error fetching requests for user %d: %v", req.UserId, err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch join requests")
	}

    // For each request, we also need community name. This implies a join or another fetch.
    // Let's assume for now repo returns CommunityID, and we'll need to hydrate community name (N+1)
    // Or better, modify repo.GetUserJoinRequests to also return community name.
    // For now, returning what we have.
    joinRequestDetailsList := make([]*communitypb.JoinRequestDetails, 0, len(dbRequests))
    if len(dbRequests) > 0 {
        userIDsToHydrate := []uint32{req.UserId} // Only need summary for the requester (self)
        userSummaries, _ := h.hydrateUserSummaries(ctx, userIDsToHydrate)
        userSummary := userSummaries[req.UserId]

        for _, jr := range dbRequests {
            // TODO: Fetch community name for jr.CommunityID
            joinRequestDetailsList = append(joinRequestDetailsList, &communitypb.JoinRequestDetails{
                RequestId:   uint32(jr.ID),
                CommunityId: uint32(jr.CommunityID),
                User:        userSummary, // This is the user WHOSE requests these are
                Status:      jr.Status,
                RequestedAt: timestamppb.New(jr.RequestedAt),
            })
        }
    }
    return &communitypb.GetUserJoinRequestsResponse{Requests: joinRequestDetailsList, HasMore: len(dbRequests) == limit}, nil
}
func (h *CommunityHandler) GetCommunityPendingRequests(ctx context.Context, req *communitypb.GetCommunityPendingRequestsRequest) (*communitypb.GetCommunityPendingRequestsResponse, error) {
    log.Printf("GetCommunityPendingRequests for Community %d by Actor %d", req.CommunityId, req.ActorUserId)
    if req.CommunityId == 0 || req.ActorUserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Community ID and Actor User ID are required")
	}

    // TODO: Permission Check - Is ActorUserID a mod/owner of CommunityID?
    // role, err := h.repo.GetUserRoleInCommunity(ctx, uint(req.CommunityId), uint(req.ActorUserId))
    // if err != nil || (role != "owner" && role != "moderator") {
    //  return nil, status.Errorf(codes.PermissionDenied, "Actor does not have permission to view pending requests")
    // }

    limit, offset := getLimitOffsetCommunity(req.Page, req.Limit)
    dbRequests, err := h.repo.GetPendingJoinRequestsForCommunity(ctx, uint(req.CommunityId), limit, offset)
    if err != nil {
		log.Printf("GetCommunityPendingRequests: Error fetching requests for community %d: %v", req.CommunityId, err)
	}

    joinRequestDetailsList := make([]*communitypb.JoinRequestDetails, 0, len(dbRequests))
    if len(dbRequests) > 0 {
        userIDsToHydrate := make([]uint32, 0, len(dbRequests))
        for _, jr := range dbRequests { userIDsToHydrate = append(userIDsToHydrate, uint32(jr.UserID)) }

        userSummaries, errHydrate := h.hydrateUserSummaries(ctx, userIDsToHydrate)
        if errHydrate != nil { log.Printf("Error hydrating users for pending requests: %v", errHydrate)}

        for _, jr := range dbRequests {
            var userSummary *communitypb.UserSummary
            if userSummaries != nil { userSummary = userSummaries[uint32(jr.UserID)] }
            if userSummary == nil { userSummary = &communitypb.UserSummary{Id: uint32(jr.UserID), Name: "Unknown User"}}

            joinRequestDetailsList = append(joinRequestDetailsList, &communitypb.JoinRequestDetails{
                RequestId:   uint32(jr.ID),
                CommunityId: uint32(jr.CommunityID),
                User:        userSummary,
                Status:      jr.Status,
                RequestedAt: timestamppb.New(jr.RequestedAt),
            })
        }
    }
    return &communitypb.GetCommunityPendingRequestsResponse{Requests: joinRequestDetailsList, HasMore: len(dbRequests) == limit}, nil
}

func (h *CommunityHandler) UpdateMemberRole(ctx context.Context, req *communitypb.UpdateMemberRoleRequest) (*emptypb.Empty, error) {
	log.Printf("UpdateMemberRole: Comm %d, Actor %d, Target %d, NewRole %s",
		req.CommunityId, req.ActorUserId, req.TargetUserId, req.NewRole)

	if req.CommunityId == 0 || req.ActorUserId == 0 || req.TargetUserId == 0 || req.NewRole == "" {
		return nil, status.Errorf(codes.InvalidArgument, "CommunityID, ActorID, TargetUserID, and NewRole are required")
	}
	if req.ActorUserId == req.TargetUserId {
		return nil, status.Errorf(codes.InvalidArgument, "Cannot change your own role via this method")
	}

	// 1. Permission Check: Actor must be an owner of the community
	actorRole, err := h.repo.GetUserRoleInCommunity(ctx, uint(req.CommunityId), uint(req.ActorUserId))
	if err != nil {
		log.Printf("UpdateMemberRole: Error getting actor's role: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to verify permissions")
	}
	if actorRole != "owner" {
		log.Printf("UpdateMemberRole: Actor %d is not an owner of community %d (role: %s)", req.ActorUserId, req.CommunityId, actorRole)
		return nil, status.Errorf(codes.PermissionDenied, "Only community owners can change member roles")
	}

	// 2. Call repository to update role
	err = h.repo.UpdateMemberRole(ctx, uint(req.CommunityId), uint(req.TargetUserId), req.NewRole)
	if err != nil {
		log.Printf("UpdateMemberRole: Error updating role in repo: %v", err)
		if strings.Contains(err.Error(), "not a member") || strings.Contains(err.Error(), "cannot change the role of an owner") || strings.Contains(err.Error(), "invalid role specified") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Failed to update member role")
	}

	// TODO: Publish "member_role_updated" event to RabbitMQ for real-time updates or audit logs
	log.Printf("Role updated for user %d in community %d to '%s' by owner %d", req.TargetUserId, req.CommunityId, req.NewRole, req.ActorUserId)
	return &emptypb.Empty{}, nil
}


// --- Helper Functions ---
func mapDBCommunityToProtoDetails(c *postgres.Community, creatorSummary *communitypb.UserSummary, memberCount int64) *communitypb.CommunityDetails {
	if c == nil { return nil }
	return &communitypb.CommunityDetails{
		Id:          uint32(c.ID),
		Name:        c.Name,
		Description: c.Description,
		CreatorId:   uint32(c.CreatorID),
		CreatorSummary: creatorSummary,
		IconUrl:     c.IconURL,
		BannerUrl:   c.BannerURL,
		Categories:  c.Categories,
		Rules:       c.Rules,
		Status:      mapDBStatusToProtoStatus(c.Status),
		CreatedAt:   timestamppb.New(c.CreatedAt),
		MemberCount: int32(memberCount),
	}
}

func mapDBCommunityToListProto(c *postgres.Community, memberCount int64, isJoined bool, hasPending bool) *communitypb.Community {
	if c == nil { return nil }
	return &communitypb.Community{ // Maps to the 'Community' message in proto, not 'CommunityDetails'
		Id:                         uint32(c.ID),
		Name:                       c.Name,
		DescriptionSnippet:         firstNCharsCommunity(c.Description, 100),
		IconUrl:                    c.IconURL,
		Status:                     mapDBStatusToProtoStatus(c.Status),
		MemberCount:                int32(memberCount),
		IsJoinedByRequester:        isJoined,
		HasPendingRequestByRequester: hasPending,
		Categories:                 c.Categories,
	}
}


func mapDBStatusToProtoStatus(dbStatus string) communitypb.CommunityStatus {
	switch strings.ToLower(dbStatus) {
	case "pending_approval":
		return communitypb.CommunityStatus_PENDING_APPROVAL
	case "active":
		return communitypb.CommunityStatus_ACTIVE
	case "rejected":
		return communitypb.CommunityStatus_REJECTED
	case "banned":
		return communitypb.CommunityStatus_BANNED
	default:
		return communitypb.CommunityStatus_COMMUNITY_STATUS_UNSPECIFIED
	}
}

func (h *CommunityHandler) hydrateUserSummaries(ctx context.Context, userIDs []uint32) (map[uint32]*communitypb.UserSummary, error) {
	if len(userIDs) == 0 || h.userClient == nil {
		return make(map[uint32]*communitypb.UserSummary), nil
	}
	// Use userClient to fetch profiles
	profilesResp, err := h.userClient.GetUserProfilesByIds(ctx, &userpb.GetUserProfilesByIdsRequest{UserIds: userIDs})
	if err != nil { return nil, fmt.Errorf("failed to fetch user profiles for summaries: %w", err) }

	summaries := make(map[uint32]*communitypb.UserSummary)
	if profilesResp != nil && profilesResp.Users != nil {
		for uid, profile := range profilesResp.Users {
			if profile != nil {
				summaries[uid] = &communitypb.UserSummary{
					Id: uid, Name: profile.Name, Username: profile.Username, ProfilePictureUrl: profile.ProfilePicture,
				}
			}
		}
	}
	return summaries, nil
}

func getLimitOffsetCommunity(page, limit int32) (int, int) {
    p := int(page); l := int(limit)
    if l <= 0 || l > 50 { l = 20 }
    if p <= 0 { p = 1 }
    return l, (p - 1) * l
}
func firstNCharsCommunity(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n] + "..."
}