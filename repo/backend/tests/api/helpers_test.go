package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"backend/internal/auth"
	"backend/internal/config"
	appErrors "backend/internal/errors"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/rbac"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------- helpers ----------

// testRouter creates a minimal Gin engine with the standard middleware
// (request ID, error handler) but WITHOUT the egress guard and WITHOUT
// the real database-backed AuthRequired middleware.
func testRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(appErrors.ErrorHandlerMiddleware())
	return r
}

// signToken creates a signed JWT for testing, using the config's JWT secret.
func signToken(userID uint, role, city, dept string, expiresIn time.Duration) string {
	cfg := config.GetConfig()
	now := time.Now()
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        fmt.Sprintf("test-jti-%d-%d", userID, now.UnixNano()),
			Issuer:    cfg.JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		},
		Role:            role,
		CityScope:       city,
		DepartmentScope: dept,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(cfg.JWTSecret))
	return tokenStr
}

// expiredToken creates an already-expired JWT.
func expiredToken(userID uint, role string) string {
	cfg := config.GetConfig()
	past := time.Now().Add(-2 * time.Hour)
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        "expired-jti",
			Issuer:    cfg.JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(past),
			ExpiresAt: jwt.NewNumericDate(past.Add(30 * time.Minute)),
		},
		Role: role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(cfg.JWTSecret))
	return tokenStr
}

// fakeAuthMiddleware injects user and claims into gin context without
// hitting a database. This is the replacement for auth.AuthRequired in tests.
func fakeAuthMiddleware() gin.HandlerFunc {
	cfg := config.GetConfig()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			appErrors.RespondUnauthorized(c, "authorization header is required")
			return
		}

		parts := splitBearer(authHeader)
		if parts == "" {
			appErrors.RespondUnauthorized(c, "invalid authorization header format")
			return
		}

		// Parse and validate the JWT.
		claims := &auth.Claims{}
		token, err := jwt.ParseWithClaims(parts, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			appErrors.RespondUnauthorized(c, "invalid or expired token")
			return
		}

		// Check for a simulated revoked token marker.
		if c.GetHeader("X-Test-Revoked") == "true" {
			appErrors.RespondUnauthorized(c, "session has been revoked")
			return
		}

		// Build user from claims.
		var uid uint
		fmt.Sscanf(claims.Subject, "%d", &uid)
		user := &models.User{
			ID:              uid,
			Username:        fmt.Sprintf("user_%d", uid),
			Role:            claims.Role,
			CityScope:       claims.CityScope,
			DepartmentScope: claims.DepartmentScope,
			Status:          "active",
		}

		// Check lockout.
		if c.GetHeader("X-Test-Locked") == "true" {
			appErrors.RespondForbidden(c, "account is temporarily locked due to too many failed login attempts")
			return
		}

		c.Set("current_user", user)
		c.Set("current_claims", claims)

		// Mirror the discrete context keys set by the real auth middleware.
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)
		c.Set("city_scope", user.CityScope)
		c.Set("dept_scope", user.DepartmentScope)

		c.Next()
	}
}

func splitBearer(header string) string {
	if len(header) > 7 && header[:7] == "Bearer " {
		return header[7:]
	}
	return ""
}

// doRequest is a convenience for making HTTP requests in tests.
func doRequest(r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseBody decodes JSON response body into a map.
func parseBody(w *httptest.ResponseRecorder) map[string]interface{} {
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body
}

// ---------- route setup helpers ----------

// setupProtectedRoute registers a single protected endpoint for testing
// RBAC and auth scenarios.
func setupProtectedRoute(method, path string, roles []string, handler gin.HandlerFunc) *gin.Engine {
	r := testRouter()
	protected := r.Group("")
	protected.Use(fakeAuthMiddleware())
	if len(roles) > 0 {
		protected.Use(rbac.RequireRole(roles...))
	}
	switch method {
	case "GET":
		protected.GET(path, handler)
	case "POST":
		protected.POST(path, handler)
	case "PUT":
		protected.PUT(path, handler)
	case "DELETE":
		protected.DELETE(path, handler)
	}
	return r
}

// dummyHandler returns a 200 OK with a simple message.
func dummyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

// dummyCreatedHandler returns 201 Created.
func dummyCreatedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": 1}})
	}
}
