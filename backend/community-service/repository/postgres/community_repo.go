package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lib/pq" // For string arrays
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	// "github.com/jackc/pgx/v5/pgconn" // If needed for specific pg errors
)

// Community model
type Community struct {
	ID          uint           `gorm:"primaryKey"`
	Name        string         `gorm:"type:varchar(100);uniqueIndex;not null"` // Unique name for community
	Description string         `gorm:"type:text"`
	CreatorID   uint           `gorm:"not null;index"`
	IconURL     string         `gorm:"type:varchar(255)"` // URL from Media Service
	BannerURL   string         `gorm:"type:varchar(255)"` // URL from Media Service
	Categories  pq.StringArray `gorm:"type:text[]"`       // PostgreSQL array of strings
	Rules       pq.StringArray `gorm:"type:text[]"`       // PostgreSQL array of strings
	Status      string         `gorm:"type:varchar(20);default:'pending_approval';not null"` // pending_approval, active, rejected, banned
	CreatedAt   time.Time      `gorm:"default:current_timestamp"`
	UpdatedAt   time.Time      `gorm:"default:current_timestamp"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// CommunityMember model
type CommunityMember struct {
	CommunityID uint      `gorm:"primaryKey;autoIncrement:false"`
	UserID      uint      `gorm:"primaryKey;autoIncrement:false"`
	Role        string    `gorm:"type:varchar(20);not null;default:'member'"` // member, moderator, owner
	JoinedAt    time.Time `gorm:"default:current_timestamp"`
	// Community   Community `gorm:"foreignKey:CommunityID"` // GORM relations
	// User        User      `gorm:"foreignKey:UserID"`      // (User model would be a simplified version here or fetched via User Service)
}

// CommunityJoinRequest model
type CommunityJoinRequest struct {
	ID          uint      `gorm:"primaryKey"`
	CommunityID uint      `gorm:"not null;index:idx_community_user_join_request,unique"`
	UserID      uint      `gorm:"not null;index:idx_community_user_join_request,unique"`
	Status      string    `gorm:"type:varchar(20);default:'pending';not null"` // pending, accepted, rejected
	RequestedAt time.Time `gorm:"default:current_timestamp"`
	ResolvedAt  *time.Time
	ResolvedBy  *uint // User ID of moderator/owner who resolved it
}

func (Community) TableName() string            { return "communities" }
func (CommunityMember) TableName() string      { return "community_members" }
func (CommunityJoinRequest) TableName() string { return "community_join_requests" }

type CommunityRepository struct{ db *gorm.DB }

func NewCommunityRepository() (*CommunityRepository, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { log.Fatalln("DATABASE_URL not set for community service") }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { return nil, fmt.Errorf("failed to connect community database: %w", err) }
	if err := db.AutoMigrate(&Community{}, &CommunityMember{}, &CommunityJoinRequest{}); err != nil {
		return nil, fmt.Errorf("failed to migrate community database: %w", err)
	}
	return &CommunityRepository{db: db}, nil
}

// CreateCommunity creates a new community and sets the creator as the owner.
func (r *CommunityRepository) CreateCommunity(ctx context.Context, community *Community) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create the community record
		// Status defaults to 'pending_approval'
		if community.Status == "" {
			community.Status = "pending_approval" // Explicitly set for clarity
		}
		if err := tx.Create(community).Error; err != nil {
			// Handle unique constraint violation for Name
			// if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "communities_name_key") {
			//  return errors.New("community name already exists")
			// }
			if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "communities_name_key") {
                 return errors.New("community name already exists")
            }
			return fmt.Errorf("failed to create community record: %w", err)
		}

		// 2. Add the creator as the owner
		ownerMember := CommunityMember{
			CommunityID: community.ID,
			UserID:      community.CreatorID,
			Role:        "owner",
		}
		if err := tx.Create(&ownerMember).Error; err != nil {
			return fmt.Errorf("failed to add creator as owner: %w", err)
		}
		log.Printf("Community '%s' (ID: %d) created by user %d, pending approval. Creator set as owner.", community.Name, community.ID, community.CreatorID)
		return nil
	})
}

func (r *CommunityRepository) CheckHealth(ctx context.Context) error {
	sqlDB, _ := r.db.DB(); return sqlDB.PingContext(ctx)
}

// GetCommunityByID retrieves a community by its ID.
func (r *CommunityRepository) GetCommunityByID(ctx context.Context, communityID uint) (*Community, error) {
	var community Community
	if err := r.db.WithContext(ctx).First(&community, communityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, errors.New("community not found") }
		return nil, fmt.Errorf("error finding community %d: %w", communityID, err)
	}
	return &community, nil
}

// GetUserRoleInCommunity checks the role of a user in a community.
func (r *CommunityRepository) GetUserRoleInCommunity(ctx context.Context, communityID, userID uint) (string, error) {
	if userID == 0 { return "none", nil } // Unauthenticated user has no role
	var member CommunityMember
	err := r.db.WithContext(ctx).
		Where("community_id = ? AND user_id = ?", communityID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Check if there's a pending join request
			var jrCount int64
			countErr := r.db.WithContext(ctx).Model(&CommunityJoinRequest{}).
				Where("community_id = ? AND user_id = ? AND status = 'pending'", communityID, userID).
				Count(&jrCount).Error
			if countErr != nil {
				log.Printf("Error counting pending join requests for user %d in community %d: %v", userID, communityID, countErr)
				return "none", nil
			}
			if jrCount > 0 { return "pending_join", nil }
			return "none", nil // Not a member, no pending request
		}
		return "none", fmt.Errorf("error checking member role: %w", err)
	}
	return member.Role, nil
}


// ListCommunitiesParams holds parameters for listing communities
type ListCommunitiesParams struct {
	Limit           int
	Offset          int
	FilterType      string   // "ALL_PUBLIC", "JOINED_BY_USER", "CREATED_BY_USER"
	UserIDContext   *uint    // For JOINED_BY_USER or CREATED_BY_USER
	SearchQuery     *string
	CategoryFilters []string
}

// ListCommunities retrieves communities based on filters.
func (r *CommunityRepository) ListCommunities(ctx context.Context, params ListCommunitiesParams) ([]Community, error) {
	var communities []Community
	query := r.db.WithContext(ctx).Model(&Community{})

	switch params.FilterType {
	case "ALL_PUBLIC":
		query = query.Where("status = ?", "active") // Assuming only active communities are public for listing
	case "JOINED_BY_USER":
		if params.UserIDContext == nil || *params.UserIDContext == 0 {
			return nil, errors.New("user_id_context is required for JOINED_BY_USER filter")
		}
		query = query.Joins("JOIN community_members cm ON cm.community_id = communities.id").
			Where("cm.user_id = ? AND communities.status = ?", *params.UserIDContext, "active")
	case "CREATED_BY_USER":
		if params.UserIDContext == nil || *params.UserIDContext == 0 {
			return nil, errors.New("user_id_context is required for CREATED_BY_USER filter")
		}
		query = query.Where("creator_id = ?", *params.UserIDContext) // Can show pending ones they created
	default: // Fallback to ALL_PUBLIC
		query = query.Where("status = ?", "active")
	}

	if params.SearchQuery != nil && *params.SearchQuery != "" {
		searchTerm := "%" + strings.ToLower(*params.SearchQuery) + "%"
		query = query.Where("LOWER(name) ILIKE ? OR LOWER(description) ILIKE ?", searchTerm, searchTerm)
	}

	if len(params.CategoryFilters) > 0 {
		// This query for array containment can be tricky with GORM directly.
		// pq.StringArray for GORM means you can often use GORM's JSON/array operators if configured,
		// or fall back to raw SQL for array operations like @> (contains).
		// Example for PostgreSQL: query = query.Where("categories @> ?", pq.Array(params.CategoryFilters))
		// For simplicity, let's assume a simple AND for categories (all must match if multiple provided)
		for _, cat := range params.CategoryFilters {
			catTerm := "%" + cat + "%" // Simple ILIKE matching for individual category terms within the array
			query = query.Where("array_to_string(categories, ',') ILIKE ?", catTerm)
		}
	}

	err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Offset).Find(&communities).Error
	return communities, err
}


// CreateJoinRequest creates a new join request or updates an existing rejected one.
func (r *CommunityRepository) CreateJoinRequest(ctx context.Context, communityID, userID uint) error {
	// 1. Check if user is already a member
	isMember, _ := r.GetUserRoleInCommunity(ctx, communityID, userID)
	if isMember == "member" || isMember == "moderator" || isMember == "owner" {
		return errors.New("already a member of this community")
	}

	joinRequest := CommunityJoinRequest{
		CommunityID: communityID,
		UserID:      userID,
		Status:      "pending", // Default status
	}
	// Upsert: Create or update status to pending if it was rejected before
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "community_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "requested_at"}),
	}).Create(&joinRequest).Error

	if err != nil { return fmt.Errorf("failed to create/update join request: %w", err) }
	log.Printf("Join request created/updated for user %d to community %d", userID, communityID)
	return nil
}

// GetJoinRequestByID retrieves a single join request.
func (r *CommunityRepository) GetJoinRequestByID(ctx context.Context, requestID uint) (*CommunityJoinRequest, error) {
    var req CommunityJoinRequest
    if err := r.db.WithContext(ctx).First(&req, requestID).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) { return nil, errors.New("join request not found")}
        return nil, err
    }
    return &req, nil
}


// UpdateJoinRequestStatus updates the status of a join request and adds user as member if accepted.
func (r *CommunityRepository) UpdateJoinRequestStatus(ctx context.Context, requestID uint, newStatus string, resolvedByUserID uint) error {
	if newStatus != "accepted" && newStatus != "rejected" {
		return errors.New("invalid status for join request")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var joinRequest CommunityJoinRequest
		if err := tx.First(&joinRequest, requestID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) { return errors.New("join request not found") }
			return err
		}
		if joinRequest.Status != "pending" {
			return errors.New("join request already resolved")
		}

		updates := map[string]interface{}{
			"status":       newStatus,
			"resolved_at":  time.Now(),
			"resolved_by":  &resolvedByUserID,
		}
		if err := tx.Model(&joinRequest).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update join request status: %w", err)
		}

		if newStatus == "accepted" {
			// Add user as a member
			member := CommunityMember{
				CommunityID: joinRequest.CommunityID,
				UserID:      joinRequest.UserID,
				Role:        "member", // Default role on acceptance
			}
			// Use FirstOrCreate in case of race conditions or re-processing, though status check should prevent
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
				return fmt.Errorf("failed to add user as community member after accepting request: %w", err)
			}
			log.Printf("User %d accepted into community %d by user %d", joinRequest.UserID, joinRequest.CommunityID, resolvedByUserID)
		} else {
			log.Printf("Join request %d for user %d to community %d rejected by user %d", requestID, joinRequest.UserID, joinRequest.CommunityID, resolvedByUserID)
		}
		return nil
	})
}

// GetCommunityMembers retrieves members of a community with pagination and role filter.
func (r *CommunityRepository) GetCommunityMembers(ctx context.Context, communityID uint, roleFilter string, limit, offset int) ([]CommunityMember, error) {
	var members []CommunityMember
	query := r.db.WithContext(ctx).Where("community_id = ?", communityID)
	if roleFilter != "" && roleFilter != "all" {
		query = query.Where("role = ?", roleFilter)
	}
	err := query.Order("role ASC, joined_at ASC").Limit(limit).Offset(offset).Find(&members).Error
	return members, err
}

// GetPendingJoinRequestsForCommunity retrieves pending requests for a community (for mods/admins).
func (r *CommunityRepository) GetPendingJoinRequestsForCommunity(ctx context.Context, communityID uint, limit, offset int) ([]CommunityJoinRequest, error) {
    var requests []CommunityJoinRequest
    err := r.db.WithContext(ctx).
        Where("community_id = ? AND status = 'pending'", communityID).
        Order("requested_at ASC").Limit(limit).Offset(offset).
        Find(&requests).Error
    return requests, err
}

// GetUserJoinRequests retrieves join requests made by a specific user.
func (r *CommunityRepository) GetUserJoinRequests(ctx context.Context, userID uint, limit, offset int) ([]CommunityJoinRequest, error) {
    var requests []CommunityJoinRequest
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID). // Can be pending, accepted, or rejected
        Order("requested_at DESC").Limit(limit).Offset(offset).
        Find(&requests).Error
    return requests, err
}


// CountCommunityMembers returns the total number of members in a community.
func (r *CommunityRepository) CountCommunityMembers(ctx context.Context, communityID uint) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&CommunityMember{}).Where("community_id = ?", communityID).Count(&count).Error
    return count, err
}

func (r *CommunityRepository) GetPendingJoinRequestByUserAndCommunity(ctx context.Context, communityID, userID uint) (*CommunityJoinRequest, error) {
    var req CommunityJoinRequest
    err := r.db.WithContext(ctx).
        Where("community_id = ? AND user_id = ? AND status = 'pending'", communityID, userID).
        First(&req).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) { return nil, errors.New("join request not found") }
        return nil, err
    }
    return &req, nil
}

func (r *CommunityRepository) GetMemberCountsForMultipleCommunities(ctx context.Context, communityIDs []uint) (map[uint]int64, error) {
    if len(communityIDs) == 0 {
        return make(map[uint]int64), nil
    }
    var results []struct {
        CommunityID uint `gorm:"column:community_id"`
        Count       int64
    }
    err := r.db.WithContext(ctx).Model(&CommunityMember{}).
        Select("community_id, count(*) as count").
        Where("community_id IN ?", communityIDs).
        Group("community_id").
        Find(&results).Error

    if err != nil { return nil, fmt.Errorf("failed to get member counts: %w", err) }

    countsMap := make(map[uint]int64)
    for _, res := range results {
        countsMap[res.CommunityID] = res.Count
    }
    return countsMap, nil
}

func (r *CommunityRepository) GetUserRolesAndPendingRequestsForCommunities(ctx context.Context, userID uint, communityIDs []uint) (map[uint]string, error) {
    if userID == 0 || len(communityIDs) == 0 {
        return make(map[uint]string), nil
    }

    statuses := make(map[uint]string)

    // 1. Check for active memberships
    var members []CommunityMember
    err := r.db.WithContext(ctx).
        Where("user_id = ? AND community_id IN ?", userID, communityIDs).
        Find(&members).Error
    if err != nil { return nil, fmt.Errorf("error fetching memberships: %w", err) }

    for _, member := range members {
        statuses[member.CommunityID] = member.Role
    }

    // 2. Check for pending join requests for communities where user is not already a member
    var communityIDsToCheckForPending []uint
    for _, cid := range communityIDs {
        if _, isMember := statuses[cid]; !isMember { // Only check if not already a member
            communityIDsToCheckForPending = append(communityIDsToCheckForPending, cid)
        }
    }

    if len(communityIDsToCheckForPending) > 0 {
        var pendingRequests []CommunityJoinRequest
        err = r.db.WithContext(ctx).
            Where("user_id = ? AND community_id IN ? AND status = 'pending'", userID, communityIDsToCheckForPending).
            Find(&pendingRequests).Error
        if err != nil { return nil, fmt.Errorf("error fetching pending join requests: %w", err) }

        for _, req := range pendingRequests {
            statuses[req.CommunityID] = "pending_join"
        }
    }
    return statuses, nil
}