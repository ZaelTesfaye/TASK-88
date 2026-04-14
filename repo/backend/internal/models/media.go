package models

import (
	"time"
)

type MediaAsset struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Title        string    `gorm:"size:500;not null;index" json:"title"`
	AudioPath    string    `gorm:"size:1000" json:"audio_path"`
	CoverArtPath string    `gorm:"size:1000" json:"cover_art_path"`
	ThemeJSON    string    `gorm:"type:json" json:"theme_json"`
	LyricsLRCPath string   `gorm:"size:1000" json:"lyrics_lrc_path"`
	Duration     int       `gorm:"default:0" json:"duration"`
	MimeType     string    `gorm:"size:100" json:"mime_type"`
	FileSizeBytes int64    `gorm:"default:0" json:"file_size_bytes"`
	UploadedBy   *uint     `gorm:"index" json:"uploaded_by"`
	Uploader     *User     `gorm:"foreignKey:UploadedBy" json:"-"`
	Status       string    `gorm:"size:50;default:'active';index" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (MediaAsset) TableName() string {
	return "media_assets"
}
