package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
    gorm.Model
    Name                  string `gorm:"type:varchar(100);not null"`
    Username              string `gorm:"type:varchar(100);unique;not null"`
    Email                 string `gorm:"type:varchar(100);unique;not null"`
    PasswordHash          string `gorm:"type:varchar(255);not null"`
    Gender                string `gorm:"type:varchar(10)"`
    ProfilePicture        string `gorm:"type:varchar(255)"`
    Banner                string `gorm:"type:varchar(255)"`
    DateOfBirth           string `gorm:"type:date"`
    SecurityQuestion      string `gorm:"type:varchar(255);not null"`
    SecurityAnswerHash    string `gorm:"type:varchar(255);not null"`
    EmailVerificationCode string `gorm:"type:varchar(100);index"`
    AccountStatus         string `gorm:"type:varchar(20);default:'pending_verification';not null"`
	AccountPrivacy 	   	  string `gorm:"type:varchar(10);default:'public';not null"`
}

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

	if err := db.AutoMigrate(&User{}); err != nil {
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