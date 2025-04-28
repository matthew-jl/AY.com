package postgres

import (
	"context"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
    gorm.Model
    Name                  string `gorm:"type:varchar(100);not null"`
    Username              string `gorm:"type:varchar(100);unique;not null"`
    Email                 string `gorm:"type:varchar(100);unique;not null"`
    Password              string `gorm:"type:varchar(255);not null"`
    Gender                string `gorm:"type:varchar(10)"`
    ProfilePicture        string `gorm:"type:varchar(255)"`
    Banner                string `gorm:"type:varchar(255)"`
    DateOfBirth           string `gorm:"type:date"`
    SecurityQuestion      string `gorm:"type:varchar(255);not null"`
    SecurityAnswer        string `gorm:"type:varchar(255);not null"`
    EmailVerificationCode string `gorm:"type:varchar(100)"`
    AccountStatus         string `gorm:"type:varchar(20);default:'pending'"`
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

	// Auto-migrate the User model
	if err := db.AutoMigrate(&User{}); err != nil {
		return nil, err
	}

	return &UserRepository{db: db}, nil
}

func (r *UserRepository) CheckHealth(ctx context.Context) error {
	var count int64
	return r.db.WithContext(ctx).Model(&User{}).Count(&count).Error
}