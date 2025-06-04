package postgres

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Notification struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	Type      string    `gorm:"type:varchar(50);not null"` // "new_follower", "like", "mention", "reply"
	Message   string    `gorm:"type:text;not null"`
	IsRead    bool      `gorm:"default:false;not null"`
	EntityID  string    `gorm:"type:varchar(100);index"` // ID of the related entity: thread ID, user ID of follower
	ActorID   *uint     `gorm:"index"`
	CreatedAt time.Time `gorm:"default:current_timestamp"`
}

func (Notification) TableName() string { return "notifications" }

type NotificationRepository struct{ db *gorm.DB }

func NewNotificationRepository() (*NotificationRepository, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { log.Fatalln("DATABASE_URL not set for notification service") }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { return nil, fmt.Errorf("failed to connect notification database: %w", err) }
	if err := db.AutoMigrate(&Notification{}); err != nil {
		return nil, fmt.Errorf("failed to migrate notification database: %w", err)
	}
	return &NotificationRepository{db: db}, nil
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, notification *Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

func (r *NotificationRepository) GetNotificationsForUser(ctx context.Context, userID uint, limit, offset int, unreadOnly bool) ([]Notification, error) {
	var notifications []Notification
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if unreadOnly {
		query = query.Where("is_read = ?", false)
	}
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifications).Error
	return notifications, err
}

func (r *NotificationRepository) MarkNotificationAsRead(ctx context.Context, notificationID uint, userID uint) error {
	res := r.db.WithContext(ctx).Model(&Notification{}).Where("id = ? AND user_id = ?", notificationID, userID).Update("is_read", true)
	if res.Error != nil { return res.Error }
	if res.RowsAffected == 0 { return fmt.Errorf("notification not found or not owned by user") }
	return nil
}

func (r *NotificationRepository) MarkAllNotificationsAsRead(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND is_read = false", userID).Update("is_read", true).Error
}

func (r *NotificationRepository) GetUnreadNotificationCount(ctx context.Context, userID uint) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND is_read = false", userID).Count(&count).Error
    return count, err
}


func (r *NotificationRepository) CheckHealth(ctx context.Context) error {
	sqlDB, _ := r.db.DB(); return sqlDB.PingContext(ctx)
}