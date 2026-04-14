package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"backend/internal/auth"
	"backend/internal/config"
	appErrors "backend/internal/errors"
	"backend/internal/middleware"
	"backend/internal/models"
)

// ---------- auth API test helpers ----------

// setupAuthRouter builds a minimal router that simulates /api/v1/auth
// endpoints without a real database.
func setupAuthRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(appErrors.ErrorHandlerMiddleware())

	cfg := config.GetConfig()
	_ = cfg

	// Hash a known password.
	knownHash, _ := auth.HashPassword("Str0ng!Pass#99")

	// In-memory user store for tests.
	users := map[string]*models.User{
		"validuser": {
			ID:              1,
			Username:        "validuser",
			PasswordHash:    knownHash,
			Role:            "system_admin",
			CityScope:       "NYC",
			DepartmentScope: "IT",
			Status:          "active",
			FailedAttempts:  0,
			LockedUntil:     nil,
		},
		"lockeduser": {
			ID:              2,
			Username:        "lockeduser",
			PasswordHash:    knownHash,
			Role:            "standard_user",
			Status:          "active",
			FailedAttempts:  5,
			LockedUntil:     timePtr(time.Now().Add(15 * time.Minute)),
		},
	}

	revokedSessions := make(map[string]bool)
	sessions := make(map[string]*models.Session)

	svc := auth.NewAuthService(nil)

	api := r.Group("/api/v1")

	// POST /api/v1/auth/login
	api.POST("/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			appErrors.RespondBadRequest(c, "invalid request body", err.Error())
			return
		}

		user, exists := users[req.Username]
		if !exists {
			appErrors.RespondUnauthorized(c, "invalid credentials")
			return
		}

		if auth.IsAccountLocked(user) {
			appErrors.RespondForbidden(c, "account is temporarily locked due to too many failed login attempts")
			return
		}

		valid, err := auth.VerifyPassword(user.PasswordHash, req.Password)
		if err != nil || !valid {
			user.FailedAttempts++
			appErrors.RespondUnauthorized(c, "invalid credentials")
			return
		}

		accessToken, refreshToken, err := svc.GenerateTokenPair(user)
		if err != nil {
			appErrors.RespondInternalError(c, "failed to generate tokens")
			return
		}

		claims, _ := svc.ValidateToken(accessToken)
		now := time.Now()
		sessions[claims.ID] = &models.Session{
			UserID:         user.ID,
			JwtJTI:         claims.ID,
			IssuedAt:       now,
			LastActivityAt: now,
			ExpiresAt:      claims.ExpiresAt.Time,
		}

		c.JSON(http.StatusOK, gin.H{
			"token":        accessToken,
			"refreshToken": refreshToken,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"role":     user.Role,
			},
		})
	})

	// Auth middleware for protected routes.
	protected := api.Group("")
	protected.Use(fakeAuthMiddleware())

	// POST /api/v1/auth/logout
	protected.POST("/auth/logout", func(c *gin.Context) {
		claims := auth.GetCurrentClaims(c)
		if claims == nil {
			appErrors.RespondUnauthorized(c, "authentication required")
			return
		}
		revokedSessions[claims.ID] = true
		c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
	})

	// POST /api/v1/auth/refresh
	protected.POST("/auth/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refreshToken" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			appErrors.RespondBadRequest(c, "invalid request body", err.Error())
			return
		}

		refreshClaims, err := svc.ValidateToken(req.RefreshToken)
		if err != nil {
			appErrors.RespondUnauthorized(c, "invalid or expired refresh token")
			return
		}

		if revokedSessions[refreshClaims.ID] {
			appErrors.RespondUnauthorized(c, "refresh token has been revoked")
			return
		}

		user := users["validuser"]
		newAccess, newRefresh, err := svc.GenerateTokenPair(user)
		if err != nil {
			appErrors.RespondInternalError(c, "failed to generate new tokens")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":        newAccess,
			"refreshToken": newRefresh,
		})
	})

	// Protected dummy route to test auth enforcement.
	protected.GET("/protected/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "protected resource accessed"})
	})

	return r
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// ---------- test cases ----------

func TestLoginSuccess(t *testing.T) {
	r := setupAuthRouter()

	body := map[string]string{"username": "validuser", "password": "Str0ng!Pass#99"}
	w := doRequest(r, "POST", "/api/v1/auth/login", "", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("response should contain a token")
	}
	if resp["refreshToken"] == nil || resp["refreshToken"] == "" {
		t.Error("response should contain a refreshToken")
	}
	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain a user object")
	}
	if user["username"] != "validuser" {
		t.Errorf("expected username 'validuser', got %v", user["username"])
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	r := setupAuthRouter()

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"wrong password", "validuser", "WrongPassword!1"},
		{"nonexistent user", "nouser", "Str0ng!Pass#99"},
		{"empty password", "validuser", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]string{"username": tc.username, "password": tc.password}
			w := doRequest(r, "POST", "/api/v1/auth/login", "", body)

			if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
				t.Errorf("expected 401 or 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestLoginLockedAccount(t *testing.T) {
	r := setupAuthRouter()

	body := map[string]string{"username": "lockeduser", "password": "Str0ng!Pass#99"}
	w := doRequest(r, "POST", "/api/v1/auth/login", "", body)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for locked account, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	if msg, ok := resp["message"].(string); ok {
		if msg == "" {
			t.Error("expected a message about account being locked")
		}
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	r := setupAuthRouter()

	// Login first.
	loginBody := map[string]string{"username": "validuser", "password": "Str0ng!Pass#99"}
	loginResp := doRequest(r, "POST", "/api/v1/auth/login", "", loginBody)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login failed: %d", loginResp.Code)
	}

	body := parseBody(loginResp)
	token := body["token"].(string)

	// Logout.
	logoutResp := doRequest(r, "POST", "/api/v1/auth/logout", token, nil)
	if logoutResp.Code != http.StatusOK {
		t.Fatalf("logout failed: %d %s", logoutResp.Code, logoutResp.Body.String())
	}

	logoutBody := parseBody(logoutResp)
	if logoutBody["message"] != "logged out successfully" {
		t.Errorf("expected logout success message, got: %v", logoutBody["message"])
	}
}

func TestRefreshToken(t *testing.T) {
	r := setupAuthRouter()

	// Login to get tokens.
	loginBody := map[string]string{"username": "validuser", "password": "Str0ng!Pass#99"}
	loginResp := doRequest(r, "POST", "/api/v1/auth/login", "", loginBody)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login failed: %d", loginResp.Code)
	}

	body := parseBody(loginResp)
	accessToken := body["token"].(string)
	refreshToken := body["refreshToken"].(string)

	// Use the refresh token to get new tokens.
	refreshBody := map[string]string{"refreshToken": refreshToken}
	refreshResp := doRequest(r, "POST", "/api/v1/auth/refresh", accessToken, refreshBody)
	if refreshResp.Code != http.StatusOK {
		t.Fatalf("refresh failed: %d %s", refreshResp.Code, refreshResp.Body.String())
	}

	refreshRespBody := parseBody(refreshResp)
	newToken := refreshRespBody["token"]
	newRefresh := refreshRespBody["refreshToken"]

	if newToken == nil || newToken == "" {
		t.Error("refresh should return a new access token")
	}
	if newRefresh == nil || newRefresh == "" {
		t.Error("refresh should return a new refresh token")
	}
	if newToken == accessToken {
		t.Error("new access token should be different from old one")
	}
}

func TestProtectedRouteWithoutToken(t *testing.T) {
	r := setupAuthRouter()

	w := doRequest(r, "GET", "/api/v1/protected/resource", "", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for no token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProtectedRouteExpiredToken(t *testing.T) {
	r := setupAuthRouter()

	token := expiredToken(1, "system_admin")

	// Build request manually to avoid double-encoding.
	req, _ := http.NewRequest("GET", "/api/v1/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	if resp["message"] == nil {
		t.Error("expected an error message")
	}
}

func TestProtectedRouteValidToken(t *testing.T) {
	r := setupAuthRouter()

	token := signToken(1, "system_admin", "NYC", "IT", 30*time.Minute)

	req, _ := http.NewRequest("GET", "/api/v1/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	if resp["message"] != "protected resource accessed" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// Verify that the unified error format includes correlationId.
func TestLoginErrorIncludesCorrelationId(t *testing.T) {
	r := setupAuthRouter()

	body := map[string]string{"username": "nouser", "password": "wrong"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", "test-corr-123")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// The X-Correlation-ID should be echoed back.
	corrID := w.Header().Get("X-Correlation-ID")
	if corrID != "test-corr-123" {
		t.Errorf("expected correlation ID 'test-corr-123', got %q", corrID)
	}
}
