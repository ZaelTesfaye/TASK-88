package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"backend/internal/auth"
	appErrors "backend/internal/errors"
	"backend/internal/rbac"
)

// setupRBACRouter creates a router with various role-restricted endpoints.
func setupRBACRouter() *gin.Engine {
	r := testRouter()

	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	// Org routes: SystemAdmin only
	orgRoutes := protected.Group("/org")
	orgRoutes.Use(rbac.RequireRole(rbac.SystemAdmin))
	{
		orgRoutes.GET("/tree", dummyHandler())
		orgRoutes.GET("/nodes", dummyHandler())
		orgRoutes.POST("/nodes", dummyCreatedHandler())
	}

	// Ingestion routes: SystemAdmin and OperationsAnalyst only
	ingestionRoutes := protected.Group("/ingestion")
	ingestionRoutes.Use(rbac.RequireRole(rbac.SystemAdmin, rbac.OperationsAnalyst))
	{
		ingestionRoutes.GET("/sources", dummyHandler())
		ingestionRoutes.POST("/jobs", dummyCreatedHandler())
	}

	// Master routes: permission-based
	masterRoutes := protected.Group("/master")
	{
		masterRoutes.GET("/:entity", rbac.RequirePermission("master_data_view"), dummyHandler())
		masterRoutes.POST("/:entity", rbac.RequirePermission("master_data_crud"), dummyCreatedHandler())
	}

	// Scope-restricted endpoint.
	protected.GET("/scoped-resource", func(c *gin.Context) {
		user := auth.GetCurrentUser(c)
		if user == nil {
			appErrors.RespondUnauthorized(c, "authentication required")
			return
		}

		// Simulate checking scope against a resource in NYC/Finance.
		if !rbac.CheckObjectScope(user, "NYC", "Finance") {
			appErrors.RespondForbidden(c, "access denied: outside your organizational scope")
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "scoped resource accessed"})
	})

	return r
}

func TestAdminCanAccessOrgRoutes(t *testing.T) {
	r := setupRBACRouter()
	token := signToken(1, rbac.SystemAdmin, "NYC", "IT", 30*time.Minute)

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"GET /org/tree", "GET", "/api/v1/org/tree", http.StatusOK},
		{"GET /org/nodes", "GET", "/api/v1/org/nodes", http.StatusOK},
		{"POST /org/nodes", "POST", "/api/v1/org/nodes", http.StatusCreated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body interface{}
			if tc.method == "POST" {
				body = gin.H{"name": "Test Node"}
			}
			w := doRequest(r, tc.method, tc.path, token, body)
			if w.Code != tc.status {
				t.Errorf("expected %d, got %d: %s", tc.status, w.Code, w.Body.String())
			}
		})
	}
}

func TestDataStewardCantAccessOrgRoutes(t *testing.T) {
	r := setupRBACRouter()
	token := signToken(2, rbac.DataSteward, "NYC", "IT", 30*time.Minute)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"GET /org/tree", "GET", "/api/v1/org/tree"},
		{"GET /org/nodes", "GET", "/api/v1/org/nodes"},
		{"POST /org/nodes", "POST", "/api/v1/org/nodes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(r, tc.method, tc.path, token, nil)
			if w.Code != http.StatusForbidden {
				t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestStandardUserCantAccessIngestion(t *testing.T) {
	r := setupRBACRouter()
	token := signToken(3, rbac.StandardUser, "NYC", "IT", 30*time.Minute)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"GET /ingestion/sources", "GET", "/api/v1/ingestion/sources"},
		{"POST /ingestion/jobs", "POST", "/api/v1/ingestion/jobs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(r, tc.method, tc.path, token, nil)
			if w.Code != http.StatusForbidden {
				t.Errorf("expected 403 for standard_user on ingestion routes, got %d: %s",
					w.Code, w.Body.String())
			}
		})
	}

	// Verify OperationsAnalyst CAN access ingestion routes.
	analystToken := signToken(4, rbac.OperationsAnalyst, "NYC", "IT", 30*time.Minute)
	w := doRequest(r, "GET", "/api/v1/ingestion/sources", analystToken, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for operations_analyst on ingestion, got %d", w.Code)
	}
}

func TestObjectScopeViolation(t *testing.T) {
	r := setupRBACRouter()

	// User scoped to LAX/Engineering trying to access a NYC/Finance resource.
	token := signToken(5, rbac.DataSteward, "LAX", "Engineering", 30*time.Minute)
	w := doRequest(r, "GET", "/api/v1/scoped-resource", token, nil)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for scope violation, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	if msg, ok := resp["message"].(string); ok {
		if msg == "" {
			t.Error("expected error message about scope violation")
		}
	}

	// User scoped to NYC/Finance SHOULD succeed.
	matchingToken := signToken(6, rbac.DataSteward, "NYC", "Finance", 30*time.Minute)
	w2 := doRequest(r, "GET", "/api/v1/scoped-resource", matchingToken, nil)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 for matching scope, got %d: %s", w2.Code, w2.Body.String())
	}

	// SystemAdmin bypasses scope.
	adminToken := signToken(7, rbac.SystemAdmin, "LAX", "Engineering", 30*time.Minute)
	w3 := doRequest(r, "GET", "/api/v1/scoped-resource", adminToken, nil)
	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 for system_admin (scope bypass), got %d: %s", w3.Code, w3.Body.String())
	}
}
