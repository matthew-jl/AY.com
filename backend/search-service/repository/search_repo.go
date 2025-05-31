package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type UserSearchIndex struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string
	Username string
	Bio      string
}
func (UserSearchIndex) TableName() string { return "users" }

type ThreadSearchIndex struct {
	ID      uint   `gorm:"primaryKey"`
	Content string
	UserID  uint
}
func (ThreadSearchIndex) TableName() string { return "threads" }

type SearchRepository struct {
	userDB   *gorm.DB
	threadDB *gorm.DB
	redisClient *redis.Client
}

func NewSearchRepository() (*SearchRepository, error) {
	userDSN := os.Getenv("USER_DB_DSN")
	threadDSN := os.Getenv("THREAD_DB_DSN")
	redisAddr := os.Getenv("REDIS_ADDR")

	if userDSN == "" || threadDSN == "" {
		return nil, fmt.Errorf("database DSNs for user and thread services are required")
	}
	if redisAddr == "" {
        log.Println("Warning: REDIS_ADDR not set, Redis features will be unavailable.")
    }


	userDB, err := gorm.Open(postgres.Open(userDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user database: %w", err)
	}
	threadDB, err := gorm.Open(postgres.Open(threadDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to thread database: %w", err)
	}

    var rdb *redis.Client
    if redisAddr != "" {
        rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
        if _, err := rdb.Ping(context.Background()).Result(); err != nil {
            log.Printf("Warning: Could not connect to Redis at %s: %v. Trending hashtags will not work.", redisAddr, err)
            rdb = nil
        } else {
            log.Println("Successfully connected to Redis for Search Service.")
        }
    }

	// Enable pg_trgm extension and create trigram indexes if they don't exist.
	// User DB
	if err := userDB.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm;").Error; err != nil {
		log.Printf("Warning: Failed to ensure pg_trgm extension on user DB: %v", err)
	}
	if err := userDB.Exec("CREATE INDEX IF NOT EXISTS idx_users_name_trgm ON users USING gin (name gin_trgm_ops);").Error; err != nil {
		log.Printf("Warning: Failed to create trigram index on users.name: %v", err)
	}
	if err := userDB.Exec("CREATE INDEX IF NOT EXISTS idx_users_username_trgm ON users USING gin (username gin_trgm_ops);").Error; err != nil {
		log.Printf("Warning: Failed to create trigram index on users.username: %v", err)
	}

	// Thread DB
	if err := threadDB.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm;").Error; err != nil {
		log.Printf("Warning: Failed to ensure pg_trgm extension on thread DB: %v", err)
	}
	if err := threadDB.Exec("CREATE INDEX IF NOT EXISTS idx_threads_content_trgm ON threads USING gin (content gin_trgm_ops);").Error; err != nil {
		log.Printf("Warning: Failed to create trigram index on threads.content: %v", err)
	}

	return &SearchRepository{userDB: userDB, threadDB: threadDB, redisClient: rdb}, nil
}

func (r *SearchRepository) SearchUsers(ctx context.Context, query string, limit, offset int) ([]UserSearchIndex, error) {
	var users []UserSearchIndex
	normalizedQuery := strings.ToLower(query)

	// trigram similarity with threshold of 0.3
	err := r.userDB.WithContext(ctx).
		Where("word_similarity(name, ?) > 0.3 OR word_similarity(username, ?) > 0.3 OR word_similarity(bio, ?) > 0.3",
			normalizedQuery, normalizedQuery, normalizedQuery).
		Order(fmt.Sprintf("GREATEST(word_similarity(name, '%s'), word_similarity(username, '%s'), word_similarity(bio, '%s')) DESC", normalizedQuery, normalizedQuery, normalizedQuery)).
		Limit(limit).Offset(offset).Find(&users).Error

	if err != nil {
		log.Printf("Error searching users with trigram similarity: %v", err)
		return nil, err
	}
	return users, nil
}

func (r *SearchRepository) SearchThreads(ctx context.Context, query string, limit, offset int) ([]ThreadSearchIndex, error) {
	var threads []ThreadSearchIndex
	normalizedQuery := strings.ToLower(query)

	err := r.threadDB.WithContext(ctx).
		Where("word_similarity(content, ?) > 0.3 AND deleted_at IS NULL", normalizedQuery).
		Order(fmt.Sprintf("word_similarity(content, '%s') DESC", normalizedQuery)).
		Limit(limit).
		Offset(offset).Find(&threads).Error

	if err != nil {
		log.Printf("Error searching threads with trigram similarity: %v", err)
		return nil, err
	}
	return threads, nil
}

// --- Trending Hashtags (Redis) ---
const trendingHashtagsKey = "trending_hashtags"
const hashtagCountsKeyPrefix = "hashtag_counts:" // e.g., hashtag_counts:2023-05-20

func (r *SearchRepository) IncrementHashtagCounts(ctx context.Context, hashtags []string) {
    if r.redisClient == nil { log.Println("IncrementHashtagCounts: Redis client not available."); return }

	if len(hashtags) == 0 { return }

	pipe := r.redisClient.Pipeline()
	for _, tag := range hashtags {
        if tag != "" {
            // Increment score in a sorted set. Score is the count.
		    pipe.ZIncrBy(ctx, trendingHashtagsKey, 1, strings.ToLower(strings.TrimSpace(tag)))
        }
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Error incrementing hashtag counts in Redis: %v", err)
	}
}

func (r *SearchRepository) GetTrendingHashtags(ctx context.Context, topN int64) ([]string, error) {
    if r.redisClient == nil { return nil, errors.New("GetTrendingHashtags: Redis client not available.") }

	// Get top N hashtags by score (count)
	// ZRevRange retrieves members from a sorted set, ordered by score from high to low.
	tags, err := r.redisClient.ZRevRange(ctx, trendingHashtagsKey, 0, topN-1).Result()
	if err != nil {
		log.Printf("Error getting trending hashtags from Redis: %v", err)
		return nil, err
	}
	// The result is just a list of members (hashtags). If you need scores: ZRevRangeWithScores
	return tags, nil
}

 func (r *SearchRepository) CheckHealth(ctx context.Context) error {
    sqlUserDB, err := r.userDB.DB()
    if err != nil { return fmt.Errorf("failed to get user DB connection for health check: %w", err)}
    if err := sqlUserDB.PingContext(ctx); err != nil { return fmt.Errorf("user DB ping failed: %w", err) }

    sqlThreadDB, err := r.threadDB.DB()
    if err != nil { return fmt.Errorf("failed to get thread DB connection for health check: %w", err)}
    if err := sqlThreadDB.PingContext(ctx); err != nil { return fmt.Errorf("thread DB ping failed: %w", err) }

    if r.redisClient != nil {
        if _, err := r.redisClient.Ping(ctx).Result(); err != nil {
            return fmt.Errorf("redis ping failed: %w", err)
        }
    } else {
        log.Println("SearchRepo Health: Redis client not configured, skipping Redis check.")
    }
    return nil
 }