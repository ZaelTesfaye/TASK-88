package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appErrors "backend/internal/errors"
	"backend/internal/logging"
	"backend/internal/models"
)

const (
	contextKeyUser   = "current_user"
	contextKeyClaims = "current_claims"
)

// AuthRequired returns a Gin middleware that enforces JWT authentication.
// It extracts the Bearer token, validates the JWT, checks revocation and
// session timeouts, updates last_activity_at, and stores user/claims in context.
func AuthRequired(db *gorm.DB) gin.HandlerFunc {
	authSvc := NewAuthService(db)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			appErrors.RespondUnauthorized(c, "authorization header is required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			appErrors.RespondUnauthorized(c, "authorization header must be in the format: Bearer <token>")
			return
		}

		tokenStr := parts[1]
		if tokenStr == "" {
			appErrors.RespondUnauthorized(c, "bearer token is empty")
			return
		}

		claims, err := authSvc.ValidateToken(tokenStr)
		if err != nil {
			appErrors.RespondUnauthorized(c, "invalid or expired token")
			return
		}

		// Check if session is revoked
		revoked, err := authSvc.IsSessionRevoked(claims.ID)
		if err != nil {
			logging.Error("auth", "middleware", "failed to check session revocation")
			appErrors.RespondWithError(c, http.StatusInternalServerError, appErrors.InternalError("failed to validate session"))
			return
		}
		if revoked {
			appErrors.RespondUnauthorized(c, "session has been revoked")
			return
		}

		// Load session from database
		var session models.Session
		result := db.Where("jwt_jti = ? AND revoked_at IS NULL", claims.ID).First(&session)
		if result.Error != nil {
			appErrors.RespondUnauthorized(c, "session not found")
			return
		}

		// Check idle timeout
		if CheckIdleTimeout(&session) {
			_ = authSvc.RevokeSession(claims.ID)
			appErrors.RespondUnauthorized(c, "session expired due to inactivity")
			return
		}

		// Check absolute timeout
		if CheckAbsoluteTimeout(&session) {
			_ = authSvc.RevokeSession(claims.ID)
			appErrors.RespondUnauthorized(c, "session expired (absolute timeout)")
			return
		}

		// Update last_activity_at
		now := time.Now()
		db.Model(&models.Session{}).Where("id = ?", session.ID).Update("last_activity_at", now)

		// Load user from database
		var user models.User
		result = db.Where("id = ? AND status = ?", session.UserID, "active").First(&user)
		if result.Error != nil {
			appErrors.RespondUnauthorized(c, "user not found or inactive")
			return
		}

		// Check if the account is locked
		if IsAccountLocked(&user) {
			appErrors.RespondForbidden(c, "account is locked")
			return
		}

		// Store user and claims in context
		c.Set(contextKeyUser, &user)
		c.Set(contextKeyClaims, claims)

		// Set discrete context keys derived from the authenticated user's DB record.
		// These must NEVER come from query params or request body.
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)
		c.Set("city_scope", user.CityScope)
		c.Set("dept_scope", user.DepartmentScope)

		c.Next()
	}
}

// GetCurrentUser retrieves the authenticated user from the Gin context.
func GetCurrentUser(c *gin.Context) *models.User {
	val, exists := c.Get(contextKeyUser)
	if !exists {
		return nil
	}
	user, ok := val.(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetCurrentClaims retrieves the JWT claims from the Gin context.
func GetCurrentClaims(c *gin.Context) *Claims {
	val, exists := c.Get(contextKeyClaims)
	if !exists {
		return nil
	}
	claims, ok := val.(*Claims)
	if !ok {
		return nil
	}
	return claims
}
