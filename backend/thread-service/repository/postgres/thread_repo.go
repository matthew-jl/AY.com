package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Thread model
type Thread struct {
    ID               uint           `gorm:"primaryKey"`
    UserID           uint           `gorm:"not null;index"`
    Content          string         `gorm:"type:text"`      
    ParentThreadID   *uint          `gorm:"index"`          
    ReplyRestriction string         `gorm:"type:varchar(20);default:'everyone';not null"` // everyone, following, verified
    ScheduledAt      *time.Time     
    PostedAt         time.Time      `gorm:"not null;default:current_timestamp"`
    CommunityID      *uint          `gorm:"index"` 
    IsAdvertisement  bool           `gorm:"default:false;not null"`
    MediaIDs         pq.Int64Array  `gorm:"type:bigint[]"`
    CreatedAt        time.Time
    UpdatedAt        time.Time
    DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// Interaction model
type ThreadInteraction struct {
    ID              uint      `gorm:"primaryKey"`
    UserID          uint      `gorm:"not null;uniqueIndex:idx_user_thread_interaction"`
    ThreadID        uint      `gorm:"not null;uniqueIndex:idx_user_thread_interaction"`
    InteractionType string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_user_thread_interaction"` // 'like', 'repost', 'bookmark'
    CreatedAt       time.Time `gorm:"default:current_timestamp"`
}

type GetThreadsParams struct {
	Limit                   int
	Offset                  int
	ByUserID                *uint
	ForUsername             string
	FeedTabType             string // "posts", "replies", "media", "likes"
	LikedByUserID           *uint  // for "likes" tab
	ExcludeUserIDs          []uint
	IncludeOnlyUserIDs      []uint
	OnlyPublicOrFollowedByUserID *uint // filter: only threads public or by users followed by this user
}

func (Thread) TableName() string { return "threads" }
func (ThreadInteraction) TableName() string { return "thread_interactions" }

type ThreadRepository struct { db *gorm.DB }

func NewThreadRepository() (*ThreadRepository, error) {
     dsn := os.Getenv("DATABASE_URL")
     if dsn == "" { log.Fatalln("DATABASE_URL not set for thread service") }
     db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
     if err != nil { return nil, fmt.Errorf("failed to connect thread database: %w", err) }
     // Migrate both models
     if err := db.AutoMigrate(&Thread{}, &ThreadInteraction{}); err != nil {
         return nil, fmt.Errorf("failed to migrate thread database: %w", err)
     }
     return &ThreadRepository{db: db}, nil
}

func (r *ThreadRepository) CreateThread(ctx context.Context, thread *Thread) error {
    if thread.ScheduledAt == nil {
        thread.PostedAt = time.Now().UTC()
    } else {
        thread.PostedAt = *thread.ScheduledAt
    }
     result := r.db.WithContext(ctx).Create(thread)
     if result.Error != nil {
         return fmt.Errorf("failed to create thread: %w", result.Error)
     }
     return nil
}

func (r *ThreadRepository) GetThreadByID(ctx context.Context, id uint) (*Thread, error) {
    var thread Thread
    result := r.db.WithContext(ctx).First(&thread, id)
    if result.Error != nil {
         if errors.Is(result.Error, gorm.ErrRecordNotFound) { return nil, errors.New("thread not found") }
         return nil, fmt.Errorf("failed to get thread %d: %w", id, result.Error)
     }
     return &thread, nil
}

func (r *ThreadRepository) AddInteraction(ctx context.Context, userID, threadID uint, interactionType string) error {
    interaction := ThreadInteraction{
        UserID:          userID,
        ThreadID:        threadID,
        InteractionType: interactionType,
    }
    result := r.db.WithContext(ctx).Create(&interaction)
    if result.Error != nil {
         var pgErr *pgconn.PgError
         if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
             log.Printf("Interaction already exists: user %d, thread %d, type %s", userID, threadID, interactionType)
             return errors.New("interaction already exists")
         }
          if errors.As(result.Error, &pgErr) && pgErr.Code == "23503" {
                log.Printf("Foreign key violation on interaction: user %d, thread %d", userID, threadID)
             return errors.New("user or thread not found for interaction")
          }
         return fmt.Errorf("failed to add interaction: %w", result.Error)
     }
     return nil
}

func (r *ThreadRepository) RemoveInteraction(ctx context.Context, userID, threadID uint, interactionType string) error {
    result := r.db.WithContext(ctx).Where("user_id = ? AND thread_id = ? AND interaction_type = ?", userID, threadID, interactionType).Delete(&ThreadInteraction{})
    if result.Error != nil {
        return fmt.Errorf("failed to remove interaction: %w", result.Error)
    }
    if result.RowsAffected == 0 {
         log.Printf("No interaction found to remove: user %d, thread %d, type %s", userID, threadID, interactionType)
        return errors.New("interaction not found")
    }
    return nil
}

 func (r *ThreadRepository) CheckHealth(ctx context.Context) error {
     sqlDB, err := r.db.DB()
     if err != nil { return fmt.Errorf("failed to get underlying DB connection: %w", err) }
     err = sqlDB.PingContext(ctx)
     if err != nil { return fmt.Errorf("thread database ping failed: %w", err) }
     return nil
 }

func (r *ThreadRepository) PerformSoftDelete(ctx context.Context, threadID uint) error {
	result := r.db.WithContext(ctx).Delete(&Thread{}, threadID)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete thread %d: %w", threadID, result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("thread not found")
	}
	return nil
}

func (r *ThreadRepository) GetThreads(ctx context.Context, params GetThreadsParams) ([]Thread, error) {
	var threads []Thread
	query := r.db.WithContext(ctx).
		Order("posted_at DESC, id DESC").
		Limit(params.Limit).
		Offset(params.Offset)

	if params.ByUserID != nil && *params.ByUserID != 0 {
		switch params.FeedTabType {
		case "posts":
			query = query.Where("user_id = ? AND parent_thread_id IS NULL", *params.ByUserID)
		case "replies":
			query = query.Where("user_id = ? AND parent_thread_id IS NOT NULL", *params.ByUserID)
		case "media":
			query = query.Where("user_id = ? AND media_ids IS NOT NULL AND array_length(media_ids, 1) > 0", *params.ByUserID)
		default:
			query = query.Where("user_id = ?", *params.ByUserID) // fallback
		}
	}

	// Filter for "following" feed
	if len(params.IncludeOnlyUserIDs) > 0 {
		query = query.Where("threads.user_id IN ?", params.IncludeOnlyUserIDs)
	}

	// Exclude threads from blocked/blocking users
	if len(params.ExcludeUserIDs) > 0 {
		log.Printf("Repo: Applying ExcludeUserIDs filter: %v", params.ExcludeUserIDs) // Add this log
		query = query.Where("threads.user_id NOT IN ?", params.ExcludeUserIDs)
	}

	result := query.Find(&threads)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get threads (type: %s, exclude: %v, include: %v): %w", params.FeedTabType, params.ExcludeUserIDs, params.IncludeOnlyUserIDs, result.Error)
	}
	log.Printf("Repo: GetThreads query executed. Found %d threads before further processing.", len(threads))
	return threads, nil
}

func (r *ThreadRepository) GetInteractionCountsForThread(ctx context.Context, threadID uint) (likeCount int64, bookmarkCount int64, err error) {
	var counts []struct {
		InteractionType string
		Count           int64
	}
	result := r.db.WithContext(ctx).Model(&ThreadInteraction{}).
		Select("interaction_type, count(*) as count").
		Where("thread_id = ?", threadID).
		Group("interaction_type").
		Find(&counts)

	if result.Error != nil {
		return 0, 0, fmt.Errorf("failed to get interaction counts for thread %d: %w", threadID, result.Error)
	}

	for _, c := range counts {
		if c.InteractionType == "like" {
			likeCount = c.Count
		} else if c.InteractionType == "bookmark" {
			bookmarkCount = c.Count
		}
	}
	return likeCount, bookmarkCount, nil
}

func (r *ThreadRepository) GetInteractionCountsForMultipleThreads(ctx context.Context, threadIDs []uint) (map[uint]map[string]int64, error) {
	if len(threadIDs) == 0 {
		return make(map[uint]map[string]int64), nil
	}
	var results []struct {
		ThreadID        uint
		InteractionType string
		Count           int64
	}
	err := r.db.WithContext(ctx).Model(&ThreadInteraction{}).
		Select("thread_id, interaction_type, count(*) as count").
		Where("thread_id IN ?", threadIDs).
		Group("thread_id, interaction_type").
		Find(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get interaction counts for multiple threads: %w", err)
	}

	countsMap := make(map[uint]map[string]int64)
	for _, res := range results {
		if _, ok := countsMap[res.ThreadID]; !ok {
			countsMap[res.ThreadID] = make(map[string]int64)
		}
		countsMap[res.ThreadID][res.InteractionType] = res.Count
	}
	return countsMap, nil
}

func (r *ThreadRepository) CheckUserInteraction(ctx context.Context, userID, threadID uint, interactionType string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&ThreadInteraction{}).
		Where("user_id = ? AND thread_id = ? AND interaction_type = ?", userID, threadID, interactionType).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check user interaction: %w", result.Error)
	}
	return count > 0, nil
}

func (r *ThreadRepository) CheckUserInteractionsForMultipleThreads(ctx context.Context, userID uint, threadIDs []uint) (map[uint]map[string]bool, error) {
	if userID == 0 || len(threadIDs) == 0 {
		return make(map[uint]map[string]bool), nil
	}
	var interactions []ThreadInteraction
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND thread_id IN ?", userID, threadIDs).
		Find(&interactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to check user interactions for multiple threads: %w", err)
	}

	userInteractionsMap := make(map[uint]map[string]bool)
	for _, interaction := range interactions {
		if _, ok := userInteractionsMap[interaction.ThreadID]; !ok {
			userInteractionsMap[interaction.ThreadID] = make(map[string]bool)
		}
		userInteractionsMap[interaction.ThreadID][interaction.InteractionType] = true
	}
	return userInteractionsMap, nil
}

func (r *ThreadRepository) GetLikedThreadsByUser(ctx context.Context, userID uint, limit, offset int) ([]Thread, error) {
	var threads []Thread
	result := r.db.WithContext(ctx).
		Joins("JOIN thread_interactions ON thread_interactions.thread_id = threads.id").
		Where("thread_interactions.user_id = ? AND thread_interactions.interaction_type = ?", userID, "like").
		Order("thread_interactions.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&threads)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get liked threads for user %d: %w", userID, result.Error)
	}
	return threads, nil
}
