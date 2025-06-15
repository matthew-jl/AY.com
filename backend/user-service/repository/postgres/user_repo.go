package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// for testing
type IUserRepo interface {
	CheckHealth(ctx context.Context) error
	CreateUser(ctx context.Context, user *User, plainPassword string, plainSecurityAnswer string, verificationCode string) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	ActivateUserAccount(ctx context.Context, userID uint) error
	UpdateVerificationCode(ctx context.Context, email string, newCode string) (*User, error)
	UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error
	UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) (*User, error)
	GetUserByID(ctx context.Context, userID uint) (*User, error)
	GetUsersByIDs(ctx context.Context, userIDs []uint) ([]User, error)
	FollowUser(ctx context.Context, followerID, followedID uint) error
	UnfollowUser(ctx context.Context, followerID, followedID uint) error
	BlockUser(ctx context.Context, blockerID, blockedID uint) error
	UnblockUser(ctx context.Context, blockerID, blockedID uint) error
	GetFollowerCount(ctx context.Context, userID uint) (int64, error)
	GetFollowingCount(ctx context.Context, userID uint) (int64, error)
	GetFollowers(ctx context.Context, userID uint, limit, offset int) ([]uint, error)
	GetFollowing(ctx context.Context, userID uint, limit, offset int) ([]uint, error)
	IsFollowing(ctx context.Context, requestUserID, targetUserID uint) (bool, error)
	HasBlocked(ctx context.Context, requestUserID, targetUserID uint) (bool, error)
	IsBlockedBy(ctx context.Context, requestUserID, targetUserID uint) (bool, error)
	GetBlockedUserIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error)
	GetBlockingUserIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error)
	GetFollowingIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error)
}


type User struct {
    gorm.Model
    Name                  string `gorm:"type:varchar(100);not null"`
    Username              string `gorm:"type:varchar(100);unique;not null"`
    Email                 string `gorm:"type:varchar(100);unique;not null"`
    PasswordHash          string `gorm:"type:varchar(255);not null"`
    Gender                string `gorm:"type:varchar(10)"`
    ProfilePicture        string `gorm:"type:varchar(255);default:''"`
    Banner                string `gorm:"type:varchar(255);default:''"`
    DateOfBirth           string `gorm:"type:date"`
    SecurityQuestion      string `gorm:"type:varchar(255);not null"`
    SecurityAnswerHash    string `gorm:"type:varchar(255);not null"`
    EmailVerificationCode string `gorm:"type:varchar(100);index"`
    AccountStatus         string `gorm:"type:varchar(20);default:'pending_verification';not null"`
	AccountPrivacy 	   	  string `gorm:"type:varchar(10);default:'public';not null"`
	SubscribedToNewsletter bool   `gorm:"default:false;not null"`
	Bio				   string `gorm:"type:text"`
}

type Follow struct {
	FollowerID uint `gorm:"primaryKey;autoIncrement:false"`
	FollowedID uint `gorm:"primaryKey;autoIncrement:false"` 
	CreatedAt  time.Time
}

type Block struct {
	BlockerID uint `gorm:"primaryKey;autoIncrement:false"` 
	BlockedID uint `gorm:"primaryKey;autoIncrement:false"` 
	CreatedAt time.Time
}

func (Follow) TableName() string { return "follows" }
func (Block) TableName() string  { return "blocks" }

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository() (*UserRepository, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, os.ErrNotExist
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &Follow{}, &Block{}); err != nil {
		return nil, err
	}

	return &UserRepository{db: db}, nil
}

func (r *UserRepository) CheckHealth(ctx context.Context) error {
	var count int64
	return r.db.WithContext(ctx).Model(&User{}).Count(&count).Error
}

func (r *UserRepository) CreateUser(ctx context.Context, user *User, plainPassword string, plainSecurityAnswer string, verificationCode string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = string(hashedPassword)

	// Hash security answer too
	hashedAnswer, err := bcrypt.GenerateFromPassword([]byte(plainSecurityAnswer), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash security answer: %w", err)
	}
	user.SecurityAnswerHash = string(hashedAnswer)

	user.EmailVerificationCode = verificationCode
	user.AccountStatus = "pending_verification"
	if user.AccountPrivacy == "" {
		user.AccountPrivacy = "public"
	}

	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			return errors.New("username or email already exists")
		}
		return fmt.Errorf("failed to create user: %w", result.Error)
	}
	return nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", result.Error)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
    var user User
    result := r.db.WithContext(ctx).Where("username = ?", username).First(&user)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, errors.New("user not found by username")
        }
        return nil, fmt.Errorf("failed to get user by username %s: %w", username, result.Error)
    }
    return &user, nil
}

func (r *UserRepository) ActivateUserAccount(ctx context.Context, userID uint) error {
	result := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"account_status":          "active",
		"email_verification_code": nil, // Clear verification code
	})

	if result.Error != nil {
		return fmt.Errorf("failed to activate user account %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found for activation or already active")
	}
	log.Printf("Activated account for user ID: %d", userID)
	return nil
}

func (r *UserRepository) UpdateVerificationCode(ctx context.Context, email string, newCode string) (*User, error) {
    var user User
    result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, errors.New("user not found with this email")
        }
        return nil, fmt.Errorf("failed to find user for resend: %w", result.Error)
    }

    if user.AccountStatus == "active" {
        return nil, errors.New("account is already active")
    }

    user.EmailVerificationCode = newCode
    updateResult := r.db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
        "email_verification_code": newCode,
        "account_status": "pending_verification",
    })

    if updateResult.Error != nil {
        return nil, fmt.Errorf("failed to update verification code for user %s: %w", email, updateResult.Error)
    }
    if updateResult.RowsAffected == 0 {
        return nil, errors.New("failed to update user record for resend")
    }
    return &user, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error {
    result := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("password_hash", newPasswordHash)
    if result.Error != nil {
        return fmt.Errorf("failed to update password for user %d: %w", userID, result.Error)
    }
    if result.RowsAffected == 0 {
        return errors.New("user not found for password update")
    }
    log.Printf("Password updated successfully for user ID: %d", userID)
    return nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found for update")
		}
		return nil, fmt.Errorf("error finding user %d for update: %w", userID, err)
	}

	result := r.db.WithContext(ctx).Model(&user).Updates(updates)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			return nil, errors.New("username or email already exists")
		}
		return nil, fmt.Errorf("failed to update user %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		log.Printf("No rows affected for user update (ID: %d), possibly no change in data.", userID)
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID uint) (*User, error) {
	var user User
	result := r.db.WithContext(ctx).Where("id = ?", userID).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found by ID")
		}
		return nil, fmt.Errorf("failed to get user by ID %d: %w", userID, result.Error)
	}
	return &user, nil
}

func (r *UserRepository) GetUsersByIDs(ctx context.Context, userIDs []uint) ([]User, error) {
    var users []User
    if len(userIDs) == 0 {
        return users, nil
    }
    result := r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to get users by IDs: %w", result.Error)
    }
    return users, nil
}

func (r *UserRepository) FollowUser(ctx context.Context, followerID, followedID uint) error {
	if followerID == followedID {
		return errors.New("user cannot follow themselves")
	}
	follow := Follow{FollowerID: followerID, FollowedID: followedID}
	// FirstOrCreate -> prevent duplicate entries if already following
	result := r.db.WithContext(ctx).FirstOrCreate(&follow)
	if result.Error != nil {
		return fmt.Errorf("failed to follow user: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		log.Printf("User %d now follows User %d", followerID, followedID)
	} else {
		log.Printf("User %d already follows User %d", followerID, followedID)
	}
	return nil
}

func (r *UserRepository) UnfollowUser(ctx context.Context, followerID, followedID uint) error {
	result := r.db.WithContext(ctx).Delete(&Follow{}, "follower_id = ? AND followed_id = ?", followerID, followedID)
	if result.Error != nil {
		return fmt.Errorf("failed to unfollow user: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		log.Printf("User %d unfollowed User %d", followerID, followedID)
	} else {
		log.Printf("User %d was not following User %d, or relationship already removed", followerID, followedID)
	}
	return nil
}

func (r *UserRepository) BlockUser(ctx context.Context, blockerID, blockedID uint) error {
	if blockerID == blockedID {
		return errors.New("user cannot block themselves")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Remove BlockerID -> BlockedID follow
		if err := tx.Where("follower_id = ? AND followed_id = ?", blockerID, blockedID).Delete(&Follow{}).Error; err != nil {
			log.Printf("Error removing follow (blocker->blocked) during block: %v", err)
		}

		// 2. Remove BlockedID -> BlockerID follow
		if err := tx.Where("follower_id = ? AND followed_id = ?", blockedID, blockerID).Delete(&Follow{}).Error; err != nil {
			log.Printf("Error removing follow (blocked->blocker) during block: %v", err)
		}

		// 3. Create the block record (or do nothing if it exists)
		block := Block{BlockerID: blockerID, BlockedID: blockedID}
		result := tx.FirstOrCreate(&block)
		if result.Error != nil {
			log.Printf("Error creating/finding block record: %v", result.Error)
			return fmt.Errorf("failed to process block: %w", result.Error)
		}
		if result.RowsAffected > 0 {
			log.Printf("User %d now blocks User %d", blockerID, blockedID)
		} else {
			log.Printf("User %d already blocks User %d", blockerID, blockedID)
		}
		return nil
	})
}

func (r *UserRepository) UnblockUser(ctx context.Context, blockerID, blockedID uint) error {
	result := r.db.WithContext(ctx).Delete(&Block{}, "blocker_id = ? AND blocked_id = ?", blockerID, blockedID)
	if result.Error != nil { return fmt.Errorf("failed to unblock user: %w", result.Error) }
	if result.RowsAffected > 0 {
		log.Printf("User %d unblocked User %d", blockerID, blockedID)
	} else {
		log.Printf("User %d was not blocking User %d, or relationship already removed", blockerID, blockedID)
	}
	return nil
}

func (r *UserRepository) GetFollowerCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("followed_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *UserRepository) GetFollowingCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("follower_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *UserRepository) GetFollowers(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	var followerIDs []uint
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("followed_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Pluck("follower_id", &followerIDs).Error
	return followerIDs, err
}

func (r *UserRepository) GetFollowing(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	var followedIDs []uint
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("follower_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Pluck("followed_id", &followedIDs).Error
	return followedIDs, err
}

func (r *UserRepository) IsFollowing(ctx context.Context, requestUserID, targetUserID uint) (bool, error) {
	if requestUserID == 0 { return false, nil }
	var count int64
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("follower_id = ? AND followed_id = ?", requestUserID, targetUserID).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) HasBlocked(ctx context.Context, requestUserID, targetUserID uint) (bool, error) {
	if requestUserID == 0 { return false, nil }
	var count int64
	err := r.db.WithContext(ctx).Model(&Block{}).Where("blocker_id = ? AND blocked_id = ?", requestUserID, targetUserID).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) IsBlockedBy(ctx context.Context, requestUserID, targetUserID uint) (bool, error) {
	if requestUserID == 0 { return false, nil }
	var count int64
	err := r.db.WithContext(ctx).Model(&Block{}).Where("blocker_id = ? AND blocked_id = ?", targetUserID, requestUserID).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) GetBlockedUserIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	var blockedIDs []uint
	err := r.db.WithContext(ctx).Model(&Block{}).Where("blocker_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Pluck("blocked_id", &blockedIDs).Error
	return blockedIDs, err
}

func (r *UserRepository) GetBlockingUserIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	var blockerIDs []uint
	err := r.db.WithContext(ctx).Model(&Block{}).Where("blocked_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Pluck("blocker_id", &blockerIDs).Error
	return blockerIDs, err
}

func (r *UserRepository) GetFollowingIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
    var followedIDs []uint
    err := r.db.WithContext(ctx).Model(&Follow{}).Where("follower_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Pluck("followed_id", &followedIDs).Error
    return followedIDs, err
}