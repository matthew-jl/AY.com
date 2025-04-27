package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"type:varchar(100);unique;not null"`
	Email    string `gorm:"type:varchar(100);unique;not null"`
}