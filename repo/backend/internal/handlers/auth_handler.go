package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	"backend/internal/auth"
	appErrors "backend/internal/errors"
	"backend/internal/logging"
	"backend/internal/models"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	db      *gorm.DB
	authSvc *auth.AuthService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		db:      db,
		authSvc: auth.NewAuthService(db),
	}
}

// RegisterRoutes registers the auth routes on the given router group.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/logout", h.Logout)
		authGroup.POST("/refresh", h.Refresh)
	}
}

// loginRequest represents the expected JSON body for login.
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginResponse represents the JSON response for a successful login.
type loginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
	User         userPayload `json:"user"`
}

type userPayload struct {
	ID              uint   `json:"id"`
	Username        string `json:"username"`
	Role            string `json:"role"`
	CityScope       string `json:"city_scope"`
	DepartmentScope string `json:"department_scope"`
	Status          string `json:"status"`
}

// Login authenticates a user by username/password, issues a JWT token pair,
// creates a session, and logs the event.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	// Look up the user
	var user models.User
	result := h.db.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Log the failed attempt without a user ID
			_ = audit.LogAction(h.db, audit.AuditEntry{
				ActionType: audit.ActionFailedLogin,
				TargetType: "user",
				TargetID:   req.Username,
				IPAddress:  c.ClientIP(),
				UserAgent:  c.Request.UserAgent(),
			})
			appErrors.RespondUnauthorized(c, "invalid credentials")
			return
		}
		logging.Error("auth", "login", fmt.Sprintf("database error looking up user: %v", result.Error))
		appErrors.RespondInternalError(c, "failed to process login")
		return
	}

	// Check if account is locked
	if auth.IsAccountLocked(&user) {
		appErrors.RespondForbidden(c, "account is temporarily locked due to too many failed login attempts")
		return
	}

	// Check if user is active
	if user.Status != "active" {
		appErrors.RespondForbidden(c, "account is not active")
		return
	}

	// Verify password
	valid, err := auth.VerifyPassword(user.PasswordHash, req.Password)
	if err != nil {
		logging.Error("auth", "login", fmt.Sprintf("password verification error: %v", err))
		appErrors.RespondInternalError(c, "failed to process login")
		return
	}

	if !valid {
		_ = h.authSvc.HandleFailedLogin(&user)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &user.ID,
			ActionType:  audit.ActionFailedLogin,
			TargetType:  "user",
			TargetID:    fmt.Sprintf("%d", user.ID),
			IPAddress:   c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
		})
		appErrors.RespondUnauthorized(c, "invalid credentials")
		return
	}

	// Reset failed login attempts on success
	if err := h.authSvc.HandleSuccessfulLogin(&user); err != nil {
		logging.Error("auth", "login", fmt.Sprintf("failed to reset login attempts: %v", err))
	}

	// Generate token pair
	accessToken, refreshToken, err := h.authSvc.GenerateTokenPair(&user)
	if err != nil {
		logging.Error("auth", "login", fmt.Sprintf("failed to generate token pair: %v", err))
		appErrors.RespondInternalError(c, "failed to generate authentication tokens")
		return
	}

	// Parse claims to get JTI for session creation
	claims, err := h.authSvc.ValidateToken(accessToken)
	if err != nil {
		logging.Error("auth", "login", fmt.Sprintf("failed to parse generated token: %v", err))
		appErrors.RespondInternalError(c, "failed to create session")
		return
	}

	// Create session record
	now := time.Now()
	session := models.Session{
		UserID:         user.ID,
		JwtJTI:         claims.ID,
		IssuedAt:       now,
		LastActivityAt: now,
		ExpiresAt:      claims.ExpiresAt.Time,
		IPAddress:      c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	}
	if err := h.db.Create(&session).Error; err != nil {
		logging.Error("auth", "login", fmt.Sprintf("failed to create session: %v", err))
		appErrors.RespondInternalError(c, "failed to create session")
		return
	}

	// Audit log the successful login
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionLogin,
		TargetType:  "session",
		TargetID:    claims.ID,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, loginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User: userPayload{
			ID:              user.ID,
			Username:        user.Username,
			Role:            user.Role,
			CityScope:       user.CityScope,
			DepartmentScope: user.DepartmentScope,
			Status:          user.Status,
		},
	})
}

// Logout revokes the current session and logs the event.
func (h *AuthHandler) Logout(c *gin.Context) {
	claims := auth.GetCurrentClaims(c)
	if claims == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}

	user := auth.GetCurrentUser(c)

	if err := h.authSvc.RevokeSession(claims.ID); err != nil {
		logging.Error("auth", "logout", fmt.Sprintf("failed to revoke session: %v", err))
		appErrors.RespondInternalError(c, "failed to revoke session")
		return
	}

	// Audit log the logout
	var actorID *uint
	if user != nil {
		actorID = &user.ID
	}
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: actorID,
		ActionType:  audit.ActionLogout,
		TargetType:  "session",
		TargetID:    claims.ID,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// refreshRequest represents the expected JSON body for token refresh.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// Refresh validates the provided refresh token and issues a new access token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	// Validate the refresh token
	claims, err := h.authSvc.ValidateToken(req.RefreshToken)
	if err != nil {
		appErrors.RespondUnauthorized(c, "invalid or expired refresh token")
		return
	}

	// Check if the refresh token's session is revoked
	revoked, err := h.authSvc.IsSessionRevoked(claims.ID)
	if err != nil {
		logging.Error("auth", "refresh", fmt.Sprintf("failed to check revocation: %v", err))
		appErrors.RespondInternalError(c, "failed to validate refresh token")
		return
	}
	if revoked {
		appErrors.RespondUnauthorized(c, "refresh token has been revoked")
		return
	}

	// Load the user
	var user models.User
	result := h.db.Where("id = ? AND status = ?", claims.Subject, "active").First(&user)
	if result.Error != nil {
		appErrors.RespondUnauthorized(c, "user not found or inactive")
		return
	}

	// Generate a new token pair
	accessToken, refreshToken, err := h.authSvc.GenerateTokenPair(&user)
	if err != nil {
		logging.Error("auth", "refresh", fmt.Sprintf("failed to generate token pair: %v", err))
		appErrors.RespondInternalError(c, "failed to generate new tokens")
		return
	}

	// Parse the new access token to get the JTI
	newClaims, err := h.authSvc.ValidateToken(accessToken)
	if err != nil {
		logging.Error("auth", "refresh", fmt.Sprintf("failed to parse generated token: %v", err))
		appErrors.RespondInternalError(c, "failed to create session")
		return
	}

	// Create a new session for the new access token
	now := time.Now()
	session := models.Session{
		UserID:         user.ID,
		JwtJTI:         newClaims.ID,
		IssuedAt:       now,
		LastActivityAt: now,
		ExpiresAt:      newClaims.ExpiresAt.Time,
		IPAddress:      c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	}
	if err := h.db.Create(&session).Error; err != nil {
		logging.Error("auth", "refresh", fmt.Sprintf("failed to create session: %v", err))
		appErrors.RespondInternalError(c, "failed to create session")
		return
	}

	// Revoke the old refresh session (if it existed as a session)
	_ = h.authSvc.RevokeSession(claims.ID)

	c.JSON(http.StatusOK, gin.H{
		"token":        accessToken,
		"refreshToken": refreshToken,
	})
}
