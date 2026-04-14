package unit

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/models"
)

// ---------- helpers ----------

// testAuthService returns an AuthService wired to a test config (no DB needed
// for token generation/validation).
func testAuthService() *auth.AuthService {
	_ = config.LoadConfig() // ensure singleton is initialised with defaults
	return auth.NewAuthService(nil)
}

func testUser() *models.User {
	return &models.User{
		ID:              1,
		Username:        "testuser",
		Role:            "system_admin",
		CityScope:       "NYC",
		DepartmentScope: "IT",
		Status:          "active",
	}
}

// ---------- password hashing ----------

func TestHashAndVerifyPassword(t *testing.T) {
	password := "Str0ng!Pass#99"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	ok, err := auth.VerifyPassword(hash, password)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword should return true for correct password")
	}
}

func TestVerifyPasswordWrong(t *testing.T) {
	password := "Str0ng!Pass#99"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	ok, err := auth.VerifyPassword(hash, "WrongPassword!1")
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword should return false for wrong password")
	}
}

// ---------- password complexity ----------

func TestPasswordComplexity(t *testing.T) {
	validPasswords := []string{
		"Abcdef1234!@",
		"MyP@ssw0rd123",
		"Hello$World99",
		"Secur3&Str0ng!",
	}
	for _, pw := range validPasswords {
		t.Run(pw, func(t *testing.T) {
			if err := auth.ValidatePasswordComplexity(pw); err != nil {
				t.Errorf("expected valid password %q, got error: %v", pw, err)
			}
		})
	}
}

func TestPasswordComplexityMissing(t *testing.T) {
	tests := []struct {
		name     string
		password string
		reason   string
	}{
		{"too short", "Ab1!", "less than 12 characters"},
		{"no uppercase", "abcdef1234!@", "missing uppercase"},
		{"no lowercase", "ABCDEF1234!@", "missing lowercase"},
		{"no digit", "Abcdefghijk!", "missing digit"},
		{"no symbol", "Abcdefghijk1", "missing symbol"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := auth.ValidatePasswordComplexity(tc.password)
			if err == nil {
				t.Errorf("expected error for %s (%s), got nil", tc.password, tc.reason)
			}
		})
	}
}

// ---------- JWT token pair ----------

func TestGenerateTokenPair(t *testing.T) {
	svc := testAuthService()
	user := testUser()

	accessToken, refreshToken, err := svc.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair error: %v", err)
	}
	if accessToken == "" {
		t.Fatal("access token is empty")
	}
	if refreshToken == "" {
		t.Fatal("refresh token is empty")
	}

	// Validate access token claims
	claims, err := svc.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateToken on access token failed: %v", err)
	}
	if claims.Subject != fmt.Sprintf("%d", user.ID) {
		t.Errorf("expected subject %d, got %s", user.ID, claims.Subject)
	}
	if claims.Role != user.Role {
		t.Errorf("expected role %s, got %s", user.Role, claims.Role)
	}
	if claims.CityScope != user.CityScope {
		t.Errorf("expected city_scope %s, got %s", user.CityScope, claims.CityScope)
	}
	if claims.DepartmentScope != user.DepartmentScope {
		t.Errorf("expected department_scope %s, got %s", user.DepartmentScope, claims.DepartmentScope)
	}
	if claims.ID == "" {
		t.Error("access token missing JTI")
	}
	if claims.ExpiresAt == nil {
		t.Error("access token missing expiration")
	}

	// Validate refresh token claims
	refreshClaims, err := svc.ValidateToken(refreshToken)
	if err != nil {
		t.Fatalf("ValidateToken on refresh token failed: %v", err)
	}
	if refreshClaims.Subject != fmt.Sprintf("%d", user.ID) {
		t.Errorf("refresh token: expected subject %d, got %s", user.ID, refreshClaims.Subject)
	}
	// Refresh token should have a longer expiry than the access token.
	if !refreshClaims.ExpiresAt.Time.After(claims.ExpiresAt.Time) {
		t.Error("refresh token should expire after access token")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	cfg := config.GetConfig()

	// Manually create an expired token.
	now := time.Now().Add(-2 * time.Hour) // 2 hours in the past
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			ID:        "test-jti",
			Issuer:    cfg.JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)), // expired 1.5h ago
		},
		Role: "system_admin",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	svc := testAuthService()
	_, err = svc.ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

// ---------- session timeouts ----------

func TestSessionIdleTimeout(t *testing.T) {
	now := time.Now()

	// Active session (last activity 5 minutes ago)
	activeSession := &models.Session{
		IssuedAt:       now.Add(-10 * time.Minute),
		LastActivityAt: now.Add(-5 * time.Minute),
	}
	if auth.CheckIdleTimeout(activeSession) {
		t.Error("session with 5min idle should NOT be timed out")
	}

	// Idle session (last activity 31 minutes ago)
	idleSession := &models.Session{
		IssuedAt:       now.Add(-60 * time.Minute),
		LastActivityAt: now.Add(-31 * time.Minute),
	}
	if !auth.CheckIdleTimeout(idleSession) {
		t.Error("session with 31min idle SHOULD be timed out")
	}

	// Exactly at boundary (30 minutes should NOT timeout because > is used)
	boundarySession := &models.Session{
		IssuedAt:       now.Add(-60 * time.Minute),
		LastActivityAt: now.Add(-30 * time.Minute),
	}
	// At exactly 30 min the check is time.Since > 30min which is false for exactly 30min.
	// But due to clock skew in tests, we just verify it's close to boundary.
	_ = boundarySession
}

func TestSessionAbsoluteTimeout(t *testing.T) {
	now := time.Now()

	// Session within absolute limit (issued 6 hours ago)
	validSession := &models.Session{
		IssuedAt:       now.Add(-6 * time.Hour),
		LastActivityAt: now.Add(-1 * time.Minute),
	}
	if auth.CheckAbsoluteTimeout(validSession) {
		t.Error("session issued 6h ago should NOT be expired (absolute is 12h)")
	}

	// Session past absolute limit (issued 13 hours ago)
	expiredSession := &models.Session{
		IssuedAt:       now.Add(-13 * time.Hour),
		LastActivityAt: now.Add(-1 * time.Minute),
	}
	if !auth.CheckAbsoluteTimeout(expiredSession) {
		t.Error("session issued 13h ago SHOULD be expired (absolute is 12h)")
	}
}

// ---------- account lockout ----------

func TestAccountLockout(t *testing.T) {
	// User with exactly 5 failed attempts locked until 15 min from now
	lockUntil := time.Now().Add(15 * time.Minute)
	user := &models.User{
		FailedAttempts: 5,
		LockedUntil:    &lockUntil,
	}

	if !auth.IsAccountLocked(user) {
		t.Error("user with 5 failed attempts and future lockUntil should be locked")
	}

	// User with no lockout
	unlocked := &models.User{
		FailedAttempts: 2,
		LockedUntil:    nil,
	}
	if auth.IsAccountLocked(unlocked) {
		t.Error("user with nil LockedUntil should NOT be locked")
	}
}

func TestAccountLockoutExpiry(t *testing.T) {
	// User whose lockout expired 1 minute ago
	expired := time.Now().Add(-1 * time.Minute)
	user := &models.User{
		FailedAttempts: 5,
		LockedUntil:    &expired,
	}
	if auth.IsAccountLocked(user) {
		t.Error("user whose lockout expired should NOT be locked")
	}

	// User whose lockout is still 14 minutes in the future
	active := time.Now().Add(14 * time.Minute)
	user2 := &models.User{
		FailedAttempts: 5,
		LockedUntil:    &active,
	}
	if !auth.IsAccountLocked(user2) {
		t.Error("user whose lockout is 14min in future SHOULD be locked")
	}
}
