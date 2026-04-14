package models

import (
	"time"
)

type User struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Username        string     `gorm:"uniqueIndex;size:100;not null" json:"username"`
	PasswordHash    string     `gorm:"size:255;not null" json:"-"`
	Role            string     `gorm:"size:50;not null;index" json:"role"`
	CityScope       string     `gorm:"size:255" json:"city_scope"`
	DepartmentScope string     `gorm:"size:255" json:"department_scope"`
	Status          string     `gorm:"size:20;default:'active';index" json:"status"`
	FailedAttempts  int        `gorm:"default:0" json:"failed_attempts"`
	LockedUntil     *time.Time `json:"locked_until"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
