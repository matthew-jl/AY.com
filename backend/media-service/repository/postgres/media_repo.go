package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Media struct {
	ID              uint `gorm:"primaryKey"`
	UploaderUserID  uint `gorm:"not null;index"`
	SupabasePath    string `gorm:"type:varchar(512);uniqueIndex;not null"` // Store path within bucket
	BucketName      string `gorm:"type:varchar(100);not null"`
	MimeType        string `gorm:"type:varchar(100);not null"`
	FileSize        int64  `gorm:"not null"` // Use int64 for file size
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Media) TableName() string { return "media" }

type MediaRepository struct { db *gorm.DB }

func NewMediaRepository() (*MediaRepository, error) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" { log.Fatalln("DATABASE_URL not set for media service") }
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil { return nil, fmt.Errorf("failed to connect media database: %w", err) }
    if err := db.AutoMigrate(&Media{}); err != nil { return nil, fmt.Errorf("failed to migrate media database: %w", err) }
    return &MediaRepository{db: db}, nil
}

func (r *MediaRepository) CreateMediaMetadata(ctx context.Context, media *Media) error {
    result := r.db.WithContext(ctx).Create(media)
    if result.Error != nil {
        var pgErr *pgconn.PgError
        if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" { // Unique violation
            return errors.New("media path already exists")
        }
        return fmt.Errorf("failed to create media metadata: %w", result.Error)
    }
    return nil
}

func (r *MediaRepository) GetMediaMetadataByID(ctx context.Context, id uint) (*Media, error) {
    var media Media
    result := r.db.WithContext(ctx).First(&media, id)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, errors.New("media not found")
        }
        return nil, fmt.Errorf("failed to get media metadata %d: %w", id, result.Error)
    }
    return &media, nil
}

 func (r *MediaRepository) CheckHealth(ctx context.Context) error {
     sqlDB, err := r.db.DB()
     if err != nil { return fmt.Errorf("failed to get underlying DB connection: %w", err) }
     err = sqlDB.PingContext(ctx)
     if err != nil { return fmt.Errorf("media database ping failed: %w", err) }
     return nil
 }