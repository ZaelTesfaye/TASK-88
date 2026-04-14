package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	appErrors "backend/internal/errors"
	"backend/internal/playback"
)

type PlaybackHandler struct {
	db      *gorm.DB
	service *playback.PlaybackService
}

func NewPlaybackHandler(db *gorm.DB) *PlaybackHandler {
	return &PlaybackHandler{
		db:      db,
		service: playback.NewPlaybackService(db),
	}
}

func (h *PlaybackHandler) RegisterRoutes(rg *gin.RouterGroup) {
	media := rg.Group("/media")
	{
		media.GET("", h.ListMedia)
		media.POST("", h.CreateMedia)
		media.GET("/:id", h.GetMedia)
		media.PUT("/:id", h.UpdateMedia)
		media.DELETE("/:id", h.DeleteMedia)
		media.GET("/:id/stream", h.StreamAudio)
		media.GET("/:id/cover", h.GetCoverArt)
		media.POST("/:id/lyrics/parse", h.ParseLyrics)
		media.GET("/:id/lyrics/search", h.SearchLyrics)
		media.GET("/formats/supported", h.GetSupportedFormats)
	}
}

func (h *PlaybackHandler) ListMedia(c *gin.Context) {
	var filter playback.MediaFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		appErrors.RespondBadRequest(c, "invalid query parameters", err.Error())
		return
	}
	result, err := h.service.ListMedia(filter)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *PlaybackHandler) CreateMedia(c *gin.Context) {
	var req playback.CreateMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}

	asset, err := h.service.CreateMedia(req)
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}

	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionCreate,
			TargetType:  "media_asset",
			TargetID:    fmt.Sprintf("%d", asset.ID),
		})
	}

	c.JSON(http.StatusCreated, asset)
}

func (h *PlaybackHandler) GetMedia(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid media ID", nil)
		return
	}
	asset, err := h.service.GetMedia(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			appErrors.RespondNotFound(c, "media asset not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, asset)
}

func (h *PlaybackHandler) UpdateMedia(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid media ID", nil)
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}

	asset, err := h.service.UpdateMedia(uint(id), updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			appErrors.RespondNotFound(c, "media asset not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionUpdate,
			TargetType:  "media_asset",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}

	c.JSON(http.StatusOK, asset)
}

func (h *PlaybackHandler) DeleteMedia(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid media ID", nil)
		return
	}

	if err := h.service.DeleteMedia(uint(id)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			appErrors.RespondNotFound(c, "media asset not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionDelete,
			TargetType:  "media_asset",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "media asset deleted"})
}

func (h *PlaybackHandler) StreamAudio(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid media ID", nil)
		return
	}

	asset, err := h.service.GetMedia(uint(id))
	if err != nil {
		appErrors.RespondNotFound(c, "media asset not found")
		return
	}

	if asset.AudioPath == "" {
		appErrors.RespondNotFound(c, "no audio file associated")
		return
	}

	if _, statErr := os.Stat(asset.AudioPath); os.IsNotExist(statErr) {
		appErrors.RespondNotFound(c, "audio file not found on disk")
		return
	}

	c.File(asset.AudioPath)
}

func (h *PlaybackHandler) GetCoverArt(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid media ID", nil)
		return
	}

	asset, err := h.service.GetMedia(uint(id))
	if err != nil {
		appErrors.RespondNotFound(c, "media asset not found")
		return
	}

	if asset.CoverArtPath == "" {
		appErrors.RespondNotFound(c, "no cover art associated")
		return
	}

	if _, statErr := os.Stat(asset.CoverArtPath); os.IsNotExist(statErr) {
		appErrors.RespondNotFound(c, "cover art file not found on disk")
		return
	}

	c.File(asset.CoverArtPath)
}

func (h *PlaybackHandler) ParseLyrics(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid media ID", nil)
		return
	}

	asset, err := h.service.GetMedia(uint(id))
	if err != nil {
		appErrors.RespondNotFound(c, "media asset not found")
		return
	}

	var lrcContent string

	if c.Request.Body != nil {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(c.Request.Body, 5*1024*1024))
		if readErr == nil && len(bodyBytes) > 0 {
			lrcContent = string(bodyBytes)
		}
	}

	if lrcContent == "" && asset.LyricsLRCPath != "" {
		data, readErr := os.ReadFile(asset.LyricsLRCPath)
		if readErr == nil {
			lrcContent = string(data)
		}
	}

	if lrcContent == "" {
		appErrors.RespondBadRequest(c, "no LRC content provided and no LRC file associated", nil)
		return
	}

	lines, parseErr := playback.ParseLRC(lrcContent)
	if parseErr != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "parse_error",
			"message": parseErr.Error(),
			"lines":   []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"line_count": len(lines),
		"lines":      lines,
		"lrc":        lrcContent,
	})
}

func (h *PlaybackHandler) SearchLyrics(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid media ID", nil)
		return
	}

	query := c.Query("q")
	if query == "" {
		appErrors.RespondBadRequest(c, "search query parameter 'q' is required", nil)
		return
	}

	asset, err := h.service.GetMedia(uint(id))
	if err != nil {
		appErrors.RespondNotFound(c, "media asset not found")
		return
	}

	if asset.LyricsLRCPath == "" {
		c.JSON(http.StatusOK, gin.H{"matches": []interface{}{}, "message": "no lyrics available"})
		return
	}

	data, readErr := os.ReadFile(asset.LyricsLRCPath)
	if readErr != nil {
		appErrors.RespondInternalError(c, "failed to read lyrics file")
		return
	}

	lines, parseErr := playback.ParseLRC(string(data))
	if parseErr != nil {
		c.JSON(http.StatusOK, gin.H{"matches": []interface{}{}, "message": "lyrics parse error"})
		return
	}

	matches := playback.SearchLyrics(lines, query)
	c.JSON(http.StatusOK, gin.H{"matches": matches, "total": len(matches)})
}

func (h *PlaybackHandler) GetSupportedFormats(c *gin.Context) {
	formats := h.service.GetSupportedFormats()
	c.JSON(http.StatusOK, formats)
}
