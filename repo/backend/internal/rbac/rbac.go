package rbac

import (
	"backend/internal/auth"
	appErrors "backend/internal/errors"

	"github.com/gin-gonic/gin"
)

// Role constants.
const (
	SystemAdmin       = "system_admin"
	DataSteward       = "data_steward"
	OperationsAnalyst = "operations_analyst"
	StandardUser      = "standard_user"
)

// permissionMatrix maps each role to its allowed permissions.
var permissionMatrix = map[string]map[string]bool{
	SystemAdmin: {
		"master_data_view":    true,
		"master_data_crud":    true,
		"master_data_import":  true,
		"master_data_review":  true,
		"version_draft":       true,
		"version_activate":    true,
		"analytics_view":      true,
		"analytics_kpi":       true,
		"reports_view":        true,
		"reports_manage":      true,
		"reports_download":    true,
		"ingestion_view":      true,
		"ingestion_manage":    true,
		"playback_view":       true,
		"audit_view":          true,
		"audit_manage":        true,
		"security_manage":     true,
		"org_manage":          true,
		"integration_manage":  true,
	},
	DataSteward: {
		"master_data_crud":   true,
		"master_data_import": true,
		"master_data_review": true,
		"master_data_view":   true,
		"version_draft":      true,
		"playback_view":      true,
	},
	OperationsAnalyst: {
		"master_data_view":  true,
		"analytics_view":    true,
		"analytics_kpi":     true,
		"reports_manage":    true,
		"reports_download":  true,
		"reports_view":      true,
		"ingestion_view":    true,
		"playback_view":     true,
	},
	StandardUser: {
		"master_data_view": true,
		"playback_view":    true,
		"reports_view":     true,
	},
}

// HasPermission checks whether a role has a given permission.
func HasPermission(role, permission string) bool {
	perms, ok := permissionMatrix[role]
	if !ok {
		return false
	}
	return perms[permission]
}

// RequireRole returns a middleware that ensures the current user has one of the specified roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		claims := auth.GetCurrentClaims(c)
		if claims == nil {
			appErrors.RespondUnauthorized(c, "authentication required")
			return
		}

		if !roleSet[claims.Role] {
			appErrors.RespondForbidden(c, "insufficient role privileges")
			return
		}

		c.Next()
	}
}

// RequirePermission returns a middleware that ensures the current user's role grants the specified permission.
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.GetCurrentClaims(c)
		if claims == nil {
			appErrors.RespondUnauthorized(c, "authentication required")
			return
		}

		if !HasPermission(claims.Role, permission) {
			appErrors.RespondForbidden(c, "insufficient permissions")
			return
		}

		c.Next()
	}
}

// EnforceScopeContext is a middleware that ensures scope-related context keys
// are present for non-admin users. It denies access by default if a non-admin
// user has no scope assigned.
func EnforceScopeContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := auth.GetCurrentUser(c)
		if user == nil {
			appErrors.RespondUnauthorized(c, "authentication required")
			return
		}

		// SystemAdmin bypasses scope checks
		if user.Role == SystemAdmin {
			c.Next()
			return
		}

		// Non-admin users must have at least one scope dimension assigned
		if user.CityScope == "" && user.DepartmentScope == "" {
			appErrors.RespondForbidden(c, "access denied: no organizational scope assigned")
			return
		}

		c.Next()
	}
}

// RequireScope returns a middleware that checks whether the current user's scope
// covers the target city and department.
func RequireScope(targetCity, targetDept string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := auth.GetCurrentUser(c)
		if user == nil {
			appErrors.RespondUnauthorized(c, "authentication required")
			return
		}

		if !CheckObjectScope(user, targetCity, targetDept) {
			appErrors.RespondForbidden(c, "access denied: outside your organizational scope")
			return
		}

		c.Next()
	}
}
