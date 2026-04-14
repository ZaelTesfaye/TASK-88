package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"

	"backend/internal/config"
	"backend/internal/logging"
	"backend/internal/models"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16

	accessTokenDuration  = 30 * time.Minute
	refreshTokenDuration = 12 * time.Hour

	idleTimeout    = 30 * time.Minute
	absoluteTimeout = 12 * time.Hour

	maxFailedAttempts = 5
	lockoutDuration   = 15 * time.Minute

	minPasswordLength = 12
)

// Claims represents the JWT claims for both access and refresh tokens.
type Claims struct {
	jwt.RegisteredClaims
	Role            string `json:"role"`
	CityScope       string `json:"city_scope"`
	DepartmentScope string `json:"department_scope"`
}

// AuthService provides authentication and session management operations.
type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db:  db,
		cfg: config.GetConfig(),
	}
}

// HashPassword hashes the given password using Argon2id.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash)

	return encoded, nil
}

// VerifyPassword verifies a password against an Argon2id hash.
func VerifyPassword(hash, password string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format: expected 6 parts, got %d", len(parts))
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, fmt.Errorf("failed to parse hash parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash)))

	if subtle.ConstantTimeCompare(expectedHash, computedHash) == 1 {
		return true, nil
	}
	return false, nil
}

// ValidatePasswordComplexity checks that a password meets the minimum complexity requirements.
func ValidatePasswordComplexity(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", minPasswordLength)
	}

	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSymbol = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSymbol {
		return fmt.Errorf("password must contain at least one symbol")
	}

	return nil
}

// GenerateTokenPair creates an access token and a refresh token for the given user.
func (s *AuthService) GenerateTokenPair(user *models.User) (accessToken, refreshToken string, err error) {
	jti := uuid.New().String()
	now := time.Now()

	accessClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			ID:        jti,
			Issuer:    s.cfg.JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenDuration)),
		},
		Role:            user.Role,
		CityScope:       user.CityScope,
		DepartmentScope: user.DepartmentScope,
	}

	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshJTI := uuid.New().String()
	refreshClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			ID:        refreshJTI,
			Issuer:    s.cfg.JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenDuration)),
		},
		Role:            user.Role,
		CityScope:       user.CityScope,
		DepartmentScope: user.DepartmentScope,
	}

	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ValidateToken parses and validates a JWT token string, returning the claims.
func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// RevokeSession marks a session as revoked by its JTI.
func (s *AuthService) RevokeSession(jti string) error {
	now := time.Now()
	result := s.db.Model(&models.Session{}).
		Where("jwt_jti = ? AND revoked_at IS NULL", jti).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("failed to revoke session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		logging.Warn("auth", "revoke_session", fmt.Sprintf("no active session found for JTI: %s", jti))
	}
	return nil
}

// IsSessionRevoked checks whether a session with the given JTI has been revoked.
func (s *AuthService) IsSessionRevoked(jti string) (bool, error) {
	var session models.Session
	result := s.db.Where("jwt_jti = ?", jti).First(&session)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, fmt.Errorf("failed to check session revocation: %w", result.Error)
	}
	return session.RevokedAt != nil, nil
}

// CheckIdleTimeout returns true if the session has exceeded the idle timeout (30 min).
func CheckIdleTimeout(session *models.Session) bool {
	return time.Since(session.LastActivityAt) > idleTimeout
}

// CheckAbsoluteTimeout returns true if the session has exceeded the absolute timeout (12 hours).
func CheckAbsoluteTimeout(session *models.Session) bool {
	return time.Since(session.IssuedAt) > absoluteTimeout
}

// HandleFailedLogin increments the user's failed login attempts and locks the account after 5 failures.
func (s *AuthService) HandleFailedLogin(user *models.User) error {
	user.FailedAttempts++
	updates := map[string]interface{}{
		"failed_attempts": user.FailedAttempts,
	}

	if user.FailedAttempts >= maxFailedAttempts {
		lockUntil := time.Now().Add(lockoutDuration)
		user.LockedUntil = &lockUntil
		updates["locked_until"] = lockUntil
		logging.Warn("auth", "lockout",
			fmt.Sprintf("account locked for user %s after %d failed attempts", user.Username, user.FailedAttempts))
	}

	result := s.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update failed login attempts: %w", result.Error)
	}
	return nil
}

// HandleSuccessfulLogin resets the user's failed login attempts and clears any lockout.
func (s *AuthService) HandleSuccessfulLogin(user *models.User) error {
	result := s.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"failed_attempts": 0,
		"locked_until":    nil,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to reset failed login attempts: %w", result.Error)
	}
	user.FailedAttempts = 0
	user.LockedUntil = nil
	return nil
}

// IsAccountLocked returns true if the user account is currently locked.
func IsAccountLocked(user *models.User) bool {
	if user.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*user.LockedUntil)
}
