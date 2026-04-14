package playback

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"backend/internal/models"
)

// Supported audio formats.
var supportedAudioFormats = []string{"mp3", "wav", "flac", "m4a"}

// Supported lyrics formats.
var supportedLyricsFormats = []string{"lrc"}

// SupportedFormats holds the lists of supported media formats.
type SupportedFormats struct {
	Audio  []string `json:"audio"`
	Lyrics []string `json:"lyrics"`
}

// MediaFilter holds the parameters for listing media assets.
type MediaFilter struct {
	Title    string `form:"title"`
	Status   string `form:"status"`
	MimeType string `form:"mime_type"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// PaginatedMedia holds a page of media assets plus pagination metadata.
type PaginatedMedia struct {
	Items      []models.MediaAsset `json:"items"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// CreateMediaRequest holds the data to create a new media asset.
type CreateMediaRequest struct {
	Title        string `json:"title" binding:"required"`
	AudioPath    string `json:"audio_path"`
	CoverArtPath string `json:"cover_art_path"`
	ThemeJSON    string `json:"theme_json"`
	LyricsLRCPath string `json:"lyrics_lrc_path"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type"`
	FileSizeBytes int64 `json:"file_size_bytes"`
	UploadedBy   *uint  `json:"uploaded_by"`
}

// PlaybackService provides media asset management operations.
type PlaybackService struct {
	db *gorm.DB
}

// NewPlaybackService creates a new PlaybackService.
func NewPlaybackService(db *gorm.DB) *PlaybackService {
	return &PlaybackService{db: db}
}

// ListMedia returns paginated media assets.
func (s *PlaybackService) ListMedia(filter MediaFilter) (*PaginatedMedia, error) {
	query := s.db.Model(&models.MediaAsset{})

	if filter.Title != "" {
		query = query.Where("title LIKE ?", "%"+filter.Title+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.MimeType != "" {
		query = query.Where("mime_type = ?", filter.MimeType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count media assets: %w", err)
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 50
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	offset := (filter.Page - 1) * filter.PageSize

	var items []models.MediaAsset
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to query media assets: %w", err)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &PaginatedMedia{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CreateMedia creates a new media asset.
func (s *PlaybackService) CreateMedia(req CreateMediaRequest) (*models.MediaAsset, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Validate mime type if provided.
	if req.MimeType != "" {
		if !isValidAudioMimeType(req.MimeType) {
			return nil, fmt.Errorf("unsupported audio format: %s", req.MimeType)
		}
	}

	asset := models.MediaAsset{
		Title:         req.Title,
		AudioPath:     req.AudioPath,
		CoverArtPath:  req.CoverArtPath,
		ThemeJSON:     req.ThemeJSON,
		LyricsLRCPath: req.LyricsLRCPath,
		Duration:      req.Duration,
		MimeType:      req.MimeType,
		FileSizeBytes: req.FileSizeBytes,
		UploadedBy:    req.UploadedBy,
		Status:        "active",
	}

	if err := s.db.Create(&asset).Error; err != nil {
		return nil, fmt.Errorf("failed to create media asset: %w", err)
	}

	return &asset, nil
}

// GetMedia returns a single media asset.
func (s *PlaybackService) GetMedia(id uint) (*models.MediaAsset, error) {
	var asset models.MediaAsset
	if err := s.db.First(&asset, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("media asset %d not found", id)
		}
		return nil, fmt.Errorf("failed to get media asset: %w", err)
	}
	return &asset, nil
}

// UpdateMedia updates an existing media asset.
func (s *PlaybackService) UpdateMedia(id uint, updates map[string]interface{}) (*models.MediaAsset, error) {
	var asset models.MediaAsset
	if err := s.db.First(&asset, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("media asset %d not found", id)
		}
		return nil, fmt.Errorf("failed to get media asset: %w", err)
	}

	if err := s.db.Model(&asset).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update media asset: %w", err)
	}

	// Reload.
	if err := s.db.First(&asset, id).Error; err != nil {
		return nil, fmt.Errorf("failed to reload media asset: %w", err)
	}

	return &asset, nil
}

// DeleteMedia soft-deletes a media asset by setting status to "deleted".
func (s *PlaybackService) DeleteMedia(id uint) error {
	result := s.db.Model(&models.MediaAsset{}).
		Where("id = ?", id).
		Update("status", "deleted")
	if result.Error != nil {
		return fmt.Errorf("failed to delete media asset: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("media asset %d not found", id)
	}
	return nil
}

// GetSupportedFormats returns supported audio and lyrics formats.
func (s *PlaybackService) GetSupportedFormats() *SupportedFormats {
	return &SupportedFormats{
		Audio:  supportedAudioFormats,
		Lyrics: supportedLyricsFormats,
	}
}

// IsSupportedAudioFormat checks if a file extension is a supported audio format.
func IsSupportedAudioFormat(ext string) bool {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	for _, f := range supportedAudioFormats {
		if f == ext {
			return true
		}
	}
	return false
}

// isValidAudioMimeType checks if a MIME type is a supported audio type.
func isValidAudioMimeType(mimeType string) bool {
	validMimeTypes := map[string]bool{
		"audio/mpeg":  true,
		"audio/mp3":   true,
		"audio/wav":   true,
		"audio/x-wav": true,
		"audio/flac":  true,
		"audio/x-flac": true,
		"audio/mp4":   true,
		"audio/m4a":   true,
		"audio/x-m4a": true,
		"audio/aac":   true,
	}
	return validMimeTypes[strings.ToLower(mimeType)]
}
