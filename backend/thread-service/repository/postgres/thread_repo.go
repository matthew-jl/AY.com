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
    UserID           uint           `gorm:"not null;index"` // Author
    Content          string         `gorm:"type:text"`      // Nullable if only media/poll
    ParentThreadID   *uint          `gorm:"index"`          // Pointer for NULL foreign key
    ReplyRestriction string         `gorm:"type:varchar(20);default:'everyone';not null"` // everyone, following, verified
    ScheduledAt      *time.Time     // Pointer for nullable timestamp
    PostedAt         time.Time      `gorm:"not null;default:current_timestamp"`
    CommunityID      *uint          `gorm:"index"` // Pointer for NULL foreign key
    IsAdvertisement  bool           `gorm:"default:false;not null"`
    MediaIDs         pq.Int64Array  `gorm:"type:bigint[]"` // Use PostgreSQL array to store media IDs
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
	Limit  int
	Offset int
	// Add UserID for following feed later
	// Add other filters later (e.g., community ID)
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

// CreateThread saves a new thread.
func (r *ThreadRepository) CreateThread(ctx context.Context, thread *Thread) error {
    // Default posted_at if not scheduled
    if thread.ScheduledAt == nil {
        thread.PostedAt = time.Now().UTC()
    } else {
        thread.PostedAt = *thread.ScheduledAt // Set posted_at to scheduled time if provided
    }
     result := r.db.WithContext(ctx).Create(thread)
     if result.Error != nil {
         // Handle potential errors (e.g., foreign key constraints if user/community doesn't exist)
         return fmt.Errorf("failed to create thread: %w", result.Error)
     }
     return nil
}

// GetThreadByID retrieves a thread.
func (r *ThreadRepository) GetThreadByID(ctx context.Context, id uint) (*Thread, error) {
    var thread Thread
    result := r.db.WithContext(ctx).First(&thread, id)
    if result.Error != nil {
         if errors.Is(result.Error, gorm.ErrRecordNotFound) { return nil, errors.New("thread not found") }
         return nil, fmt.Errorf("failed to get thread %d: %w", id, result.Error)
     }
     return &thread, nil
}

// AddInteraction adds a like, repost, or bookmark.
func (r *ThreadRepository) AddInteraction(ctx context.Context, userID, threadID uint, interactionType string) error {
    interaction := ThreadInteraction{
        UserID:          userID,
        ThreadID:        threadID,
        InteractionType: interactionType,
    }
    // Create will fail if unique constraint violated (already exists)
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

// RemoveInteraction removes a like, repost, or bookmark.
func (r *ThreadRepository) RemoveInteraction(ctx context.Context, userID, threadID uint, interactionType string) error {
    result := r.db.WithContext(ctx).Where("user_id = ? AND thread_id = ? AND interaction_type = ?", userID, threadID, interactionType).Delete(&ThreadInteraction{})
    if result.Error != nil {
        return fmt.Errorf("failed to remove interaction: %w", result.Error)
    }
    if result.RowsAffected == 0 {
         log.Printf("No interaction found to remove: user %d, thread %d, type %s", userID, threadID, interactionType)
        return errors.New("interaction not found") // Or return nil if non-existence is ok
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

 // PerformSoftDelete uses GORM's soft delete feature.
func (r *ThreadRepository) PerformSoftDelete(ctx context.Context, threadID uint) error {
	// GORM automatically handles setting the DeletedAt field when Delete is called
	result := r.db.WithContext(ctx).Delete(&Thread{}, threadID)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete thread %d: %w", threadID, result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("thread not found") // Return specific error
	}
	return nil
}

// GetThreads retrieves a paginated list of threads, newest first.
func (r *ThreadRepository) GetThreads(ctx context.Context, params GetThreadsParams) ([]Thread, error) {
	var threads []Thread
	query := r.db.WithContext(ctx).
		Order("posted_at DESC, id DESC"). // Order by post time (or created_at), then ID
		Limit(params.Limit).
		Offset(params.Offset)

	// Add WHERE clauses later for following feed, communities etc.
	// e.g., if params.UserID != 0: query = query.Where("user_id IN (?)", following_subquery)

	result := query.Find(&threads)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get threads: %w", result.Error)
	}
	return threads, nil
}

// Add methods for user threads, interactions etc. later