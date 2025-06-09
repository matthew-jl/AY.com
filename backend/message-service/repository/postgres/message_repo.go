package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Chat represents a conversation (direct or group)
type Chat struct {
	ID        uint      `gorm:"primaryKey"`
	Type      string    `gorm:"type:varchar(20);not null;default:'direct'"` // 'direct', 'group'
	Name      *string   `gorm:"type:varchar(100)"`                        // Group chat name
	CreatorID uint      `gorm:"not null"`                                   // User who initiated/created
	CreatedAt time.Time `gorm:"default:current_timestamp"`
	UpdatedAt time.Time `gorm:"default:current_timestamp"`
	Participants []ChatParticipant `gorm:"foreignKey:ChatID"` // GORM relation
	Messages     []Message         `gorm:"foreignKey:ChatID"` // GORM relation
}

type ChatPreview struct {
	Chat
	LastMessage     *Message          `gorm:"-"` // Loaded separately
	Participants    []ChatParticipant `gorm:"-"` // Loaded separately
	OtherUserName   string            `gorm:"-"` // For direct chats
	OtherUserPic    string            `gorm:"-"` // For direct chats
	UnreadCount     int64             `gorm:"-"` // Loaded separately
}

// ChatParticipant links users to chats
type ChatParticipant struct {
	ChatID    uint      `gorm:"primaryKey;autoIncrement:false"`
	UserID    uint      `gorm:"primaryKey;autoIncrement:false"`
	JoinedAt  time.Time `gorm:"default:current_timestamp"`
	IsHidden  bool      `gorm:"default:false;not null"`
    HiddenAt  *time.Time
	// LastReadMessageID *uint // For read receipts later
	// IsAdmin           bool   `gorm:"default:false"` // For group chats
}

// Message represents a single message in a chat
type Message struct {
	ID             uint          `gorm:"primaryKey"`
	ChatID         uint          `gorm:"not null;index"`
	SenderID       uint          `gorm:"not null;index"`
	Content        string        `gorm:"type:text"` // Nullable if only media
	Type           string        `gorm:"type:varchar(20);default:'text';not null"` // 'text', 'image', 'video', 'gif'
	MediaIDs       pq.Int64Array `gorm:"type:bigint[]"` // Array of media IDs from Media Service
	SentAt         time.Time     `gorm:"default:current_timestamp;index"`
	IsDeleted      bool          `gorm:"default:false"`
	DeletedAt      *time.Time    // For tracking when deleted
	// ReadBy         []uint    `gorm:"-"` // For read receipts, more complex, often separate table
}

func (Chat) TableName() string            { return "chats" }
func (ChatParticipant) TableName() string { return "chat_participants" }
func (Message) TableName() string         { return "messages" }

type MessageRepository struct{ db *gorm.DB }

func (r *MessageRepository) DB() *gorm.DB {
	return r.db
}

func NewMessageRepository() (*MessageRepository, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { log.Fatalln("DATABASE_URL not set for message service") }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { return nil, fmt.Errorf("failed to connect message database: %w", err) }
	if err := db.AutoMigrate(&Chat{}, &ChatParticipant{}, &Message{}); err != nil {
		return nil, fmt.Errorf("failed to migrate message database: %w", err)
	}
	return &MessageRepository{db: db}, nil
}

func (r *MessageRepository) GetOrCreateDirectChat(ctx context.Context, userID1, userID2 uint) (*Chat, error) {
	if userID1 == userID2 { return nil, errors.New("cannot create direct chat with oneself") }

	// Order user IDs to ensure consistency for finding existing chats
	u1, u2 := userID1, userID2
	if u1 > u2 { u1, u2 = u2, u1 }

	var chat Chat

	// Transaction to handle find or create logic
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Subquery to find chat_id where both users are participants and it's a direct chat with exactly 2 participants
		var existingChatID uint
		tx.Raw(`
            SELECT c.id FROM chats c
            WHERE c.type = 'direct'
            AND EXISTS (SELECT 1 FROM chat_participants cp WHERE cp.chat_id = c.id AND cp.user_id = ?)
            AND EXISTS (SELECT 1 FROM chat_participants cp WHERE cp.chat_id = c.id AND cp.user_id = ?)
            AND (SELECT COUNT(*) FROM chat_participants cp WHERE cp.chat_id = c.id) = 2
            LIMIT 1
        `, u1, u2).Scan(&existingChatID)

		if existingChatID != 0 {
			log.Printf("Found existing direct chat ID %d for users %d, %d", existingChatID, userID1, userID2)
			// Fetch the existing chat
			return tx.Preload("Participants").First(&chat, existingChatID).Error // Preload participants
		}

		// Create new chat if not found
		log.Printf("Creating new direct chat for users %d, %d", userID1, userID2)
		newChat := Chat{Type: "direct", CreatorID: userID1} // userID1 is the initiator
		if err := tx.Create(&newChat).Error; err != nil {
			return fmt.Errorf("failed to create new chat: %w", err)
		}

		participants := []ChatParticipant{
			{ChatID: newChat.ID, UserID: userID1},
			{ChatID: newChat.ID, UserID: userID2},
		}
		if err := tx.Create(&participants).Error; err != nil {
			return fmt.Errorf("failed to add participants to new chat: %w", err)
		}
		chat = newChat // Assign the newly created chat
		chat.Participants = participants // Manually assign preloaded participants
		return nil
	})

	if err != nil { return nil, err }
	return &chat, nil
}

func (r *MessageRepository) CreateGroupChat(ctx context.Context, creatorID uint, groupName string, initialParticipantIDs []uint) (*Chat, error) {
	if strings.TrimSpace(groupName) == "" { return nil, errors.New("group name cannot be empty") }

	var chat Chat
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		newChat := Chat{Type: "group", Name: &groupName, CreatorID: creatorID}
		if err := tx.Create(&newChat).Error; err != nil {
			return fmt.Errorf("failed to create group chat record: %w", err)
		}

		// Add creator as the first participant
		allParticipantStructs := []ChatParticipant{{ChatID: newChat.ID, UserID: creatorID}}
		seenParticipants := make(map[uint]bool)
		seenParticipants[creatorID] = true

		for _, pid := range initialParticipantIDs {
			if pid != 0 && pid != creatorID && !seenParticipants[pid] {
				allParticipantStructs = append(allParticipantStructs, ChatParticipant{ChatID: newChat.ID, UserID: pid})
				seenParticipants[pid] = true
			}
		}
		if len(allParticipantStructs) > 0 {
            if err := tx.Create(&allParticipantStructs).Error; err != nil {
                 return fmt.Errorf("failed to add participants to group chat: %w", err)
            }
        }


		chat = newChat
		chat.Participants = allParticipantStructs
		return nil
	})
	if err != nil { return nil, err }
    log.Printf("Group chat '%s' (ID: %d) created by user %d with %d initial participants (incl. creator)", groupName, chat.ID, creatorID, len(chat.Participants))
	return &chat, nil
}


func (r *MessageRepository) AddParticipantToGroup(ctx context.Context, chatID uint, userIDToAdd uint, addedByUserID uint) (bool, error) {
	// Validate chat exists and is a group chat
	chat, err := r.GetChatByID(ctx, chatID)
	if err != nil { return false, err }
	if chat.Type != "group" { return false, errors.New("cannot add participant to a direct chat") }

	// TODO: Validate if addedByUserID has permission (e.g., is admin or participant)

	participant := ChatParticipant{ChatID: chatID, UserID: userIDToAdd, IsHidden: false}
	// Use Clauses OnConflict to handle if user was previously in chat and "hid" it, effectively "re-joining".
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_hidden", "hidden_at"}), // Update these if conflict
	}).Create(&participant)


	if result.Error != nil {
		return false, fmt.Errorf("failed to add/update participant %d to chat %d: %w", userIDToAdd, chatID, result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("User %d added/rejoined to group chat %d by user %d", userIDToAdd, chatID, addedByUserID)
        return true, nil
	}
    log.Printf("User %d is already an active participant in group chat %d", userIDToAdd, chatID)
	return false, nil
}


func (r *MessageRepository) RemoveParticipantFromGroup(ctx context.Context, chatID uint, userIDToRemove uint, removedByUserID uint) error {
	// Validate chat exists and is a group chat
	chat, err := r.GetChatByID(ctx, chatID)
	if err != nil { return err }
	if chat.Type != "group" { return errors.New("cannot remove participant from a direct chat") }

	// TODO: Validate if removedByUserID has permission (e.g., is admin, or self-removal)
    // TODO: Ensure chat doesn't become empty or handle admin transfer if last admin leaves

	// Using "hide" logic for removal from view, actual record remains for history.
	result := r.db.WithContext(ctx).Model(&ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userIDToRemove).
		Updates(map[string]interface{}{"is_hidden": true, "hidden_at": time.Now()})


	if result.Error != nil { return fmt.Errorf("failed to remove participant %d from chat %d: %w", userIDToRemove, chatID, result.Error) }
	if result.RowsAffected == 0 { return errors.New("participant not found in chat or already removed/hidden") }

	log.Printf("User %d removed (hidden) from group chat %d by user %d", userIDToRemove, chatID, removedByUserID)
	return nil
}

func (r *MessageRepository) GetChatParticipantIDs(ctx context.Context, chatID uint) ([]uint, error) {
    var participantIDs []uint
    err := r.db.WithContext(ctx).Model(&ChatParticipant{}).Where("chat_id = ?", chatID).Pluck("user_id", &participantIDs).Error
    return participantIDs, err
}

// --- Message Management ---

func (r *MessageRepository) CreateMessage(ctx context.Context, message *Message) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create the message
		if err := tx.Create(message).Error; err != nil {
			return fmt.Errorf("failed to create message: %w", err)
		}
		// 2. Update the chat's UpdatedAt timestamp
		// Use message.SentAt if available, otherwise current time
		updateTime := message.SentAt
		if updateTime.IsZero() { updateTime = time.Now().UTC() }

		if err := tx.Model(&Chat{}).Where("id = ?", message.ChatID).Update("updated_at", updateTime).Error; err != nil {
			// Log error but don't fail message creation if this fails
			log.Printf("Warning: Failed to update chat %d UpdatedAt timestamp: %v", message.ChatID, err)
		}
		return nil
	})
}

func (r *MessageRepository) GetMessagesForChat(ctx context.Context, chatID uint, limit, offset int) ([]Message, error) {
	var messages []Message
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		// Where("chat_id = ? AND is_deleted = false", chatID).
		Order("sent_at DESC"). // Newest first
		Limit(limit).Offset(offset).
		Find(&messages).Error
	return messages, err
}

// GetUserChats retrieves ChatPreview objects for a user
func (r *MessageRepository) GetUserChats(ctx context.Context, userID uint, limit, offset int) ([]ChatPreview, error) {
    var chatPreviews []ChatPreview
    // This query gets the basic chat info and sorts by the chat's last update time
    rows, err := r.db.WithContext(ctx).
        Table("chats c").
        Select("c.id, c.type, c.name, c.creator_id, c.created_at, c.updated_at").
        Joins("JOIN chat_participants cp ON cp.chat_id = c.id").
        Where("cp.user_id = ? AND cp.is_hidden = FALSE", userID).
        Order("c.updated_at DESC").
        Limit(limit).Offset(offset).
        Rows()
    if err != nil { return nil, fmt.Errorf("failed to get user chats: %w", err)}
    defer rows.Close()

    for rows.Next() {
        var cp ChatPreview
        if err := r.db.ScanRows(rows, &cp.Chat); err != nil { // Scan into nested Chat
            log.Printf("Error scanning chat row: %v", err)
            continue
        }
        // For each chat, fetch its last message and participants (N+1 issue here, needs optimization)
        // Last Message
        var lastMsg Message
        r.db.WithContext(ctx).Where("chat_id = ?", cp.ID).Order("sent_at DESC").Limit(1).First(&lastMsg)
        if lastMsg.ID != 0 { cp.LastMessage = &lastMsg }

        // Participants (for display name / group members)
        var participantsDB []ChatParticipant
        r.db.WithContext(ctx).Where("chat_id = ?", cp.ID).Find(&participantsDB)
        cp.Participants = participantsDB // Store raw participants

        chatPreviews = append(chatPreviews, cp)
    }
    return chatPreviews, nil
}

// MarkMessageAsDeleted (soft delete)
func (r *MessageRepository) MarkMessageAsDeleted(ctx context.Context, messageID, userID uint) (*Message, error) {
	var message Message
	// First, verify the user is the sender of the message
	if err := r.db.WithContext(ctx).Where("id = ? AND sender_id = ?", messageID, userID).First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil , errors.New("message not found or not owned by user")
		}
		return nil , err
	}
	// Check if message is within deletable window (e.g., 1 minute)
	if time.Since(message.SentAt) > 1*time.Minute {
		return nil, errors.New("message can no longer be deleted (past 1 minute window)")
	}
	err := r.db.WithContext(ctx).Model(&Message{}).Where("id = ?", messageID).Updates(map[string]interface{}{
		"is_deleted": true,
		"deleted_at": time.Now(),
		"content":    "Message deleted", // Or empty string
        "media_ids":  pq.Int64Array{},   // Clear media
	}).Error

	if err != nil {
		return nil, err
	}

	return &message, nil
}

func (r *MessageRepository) CheckHealth(ctx context.Context) error {
	sqlDB, _ := r.db.DB(); return sqlDB.PingContext(ctx)
}

func (r *MessageRepository) HideChatForUser(ctx context.Context, chatID uint, userID uint) error {
	result := r.db.WithContext(ctx).Model(&ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Updates(map[string]interface{}{"is_hidden": true, "hidden_at": time.Now()})

	if result.Error != nil {
		return fmt.Errorf("failed to hide chat %d for user %d: %w", chatID, userID, result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("chat participation not found or already hidden")
	}
	log.Printf("Chat %d hidden for user %d", chatID, userID)
	return nil
}

func (r *MessageRepository) GetChatByID(ctx context.Context, chatID uint) (*Chat, error) {
    var chat Chat
    if err := r.db.WithContext(ctx).First(&chat, chatID).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("chat not found")
        }
        return nil, fmt.Errorf("failed to get chat %d: %w", chatID, err)
    }
    return &chat, nil
}

func (r *MessageRepository) IsUserParticipant(ctx context.Context, chatID, userID uint) (bool, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&ChatParticipant{}).
        Where("chat_id = ? AND user_id = ? AND is_hidden = FALSE", chatID, userID).
        Count(&count).Error
    if err != nil {
        return false, fmt.Errorf("error checking chat participation: %w", err)
    }
    return count > 0, nil
}