package unit

import (
	"testing"

	"backend/internal/models"
	"backend/internal/rbac"
)

// ---------- scope resolver (pure logic, no DB) ----------

func TestCheckObjectScopeExactMatch(t *testing.T) {
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "NYC",
		DepartmentScope: "Finance",
	}
	if !rbac.CheckObjectScope(user, "NYC", "Finance") {
		t.Error("exact match on city and department should be allowed")
	}
}

func TestCheckObjectScopeCityMismatch(t *testing.T) {
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "NYC",
		DepartmentScope: "Finance",
	}
	if rbac.CheckObjectScope(user, "LAX", "Finance") {
		t.Error("mismatched city should be denied")
	}
}

func TestCheckObjectScopeDeptMismatch(t *testing.T) {
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "NYC",
		DepartmentScope: "Finance",
	}
	if rbac.CheckObjectScope(user, "NYC", "Engineering") {
		t.Error("mismatched department should be denied")
	}
}

func TestCheckObjectScopeSystemAdminBypass(t *testing.T) {
	user := &models.User{
		Role:            rbac.SystemAdmin,
		CityScope:       "NYC",
		DepartmentScope: "Finance",
	}
	// SystemAdmin bypasses all scope restrictions.
	if !rbac.CheckObjectScope(user, "LAX", "Engineering") {
		t.Error("system_admin should bypass scope checks")
	}
}

func TestCheckObjectScopeWildcardCity(t *testing.T) {
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "*",
		DepartmentScope: "Finance",
	}
	if !rbac.CheckObjectScope(user, "LAX", "Finance") {
		t.Error("wildcard city scope should allow any city")
	}
	if !rbac.CheckObjectScope(user, "CHI", "Finance") {
		t.Error("wildcard city scope should allow any city")
	}
}

func TestCheckObjectScopeWildcardDept(t *testing.T) {
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "NYC",
		DepartmentScope: "*",
	}
	if !rbac.CheckObjectScope(user, "NYC", "Engineering") {
		t.Error("wildcard department scope should allow any department")
	}
}

func TestCheckObjectScopeBothWildcards(t *testing.T) {
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "*",
		DepartmentScope: "*",
	}
	if !rbac.CheckObjectScope(user, "LAX", "Engineering") {
		t.Error("both wildcards should allow everything")
	}
}

func TestCheckObjectScopeEmptyMeansDenied(t *testing.T) {
	// Fail-closed: non-admin with no scope is denied.
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "",
		DepartmentScope: "",
	}
	if rbac.CheckObjectScope(user, "LAX", "Engineering") {
		t.Error("empty scope for non-admin should deny access (fail-closed)")
	}
}

func TestCheckObjectScopeEmptyObjectFields(t *testing.T) {
	// When the object's city or department is empty, scope check should pass
	// because the condition skips comparison for empty object fields.
	user := &models.User{
		Role:            rbac.DataSteward,
		CityScope:       "NYC",
		DepartmentScope: "Finance",
	}
	if !rbac.CheckObjectScope(user, "", "") {
		t.Error("empty object city/dept should pass scope check")
	}
}

// ---------- permission evaluator (pure logic, no DB) ----------

func TestHasPermissionKnownRole(t *testing.T) {
	if !rbac.HasPermission(rbac.SystemAdmin, "org_manage") {
		t.Error("system_admin should have org_manage")
	}
}

func TestHasPermissionUnknownRole(t *testing.T) {
	if rbac.HasPermission("nonexistent_role", "master_data_view") {
		t.Error("unknown role should have no permissions")
	}
}

func TestHasPermissionUnknownPermission(t *testing.T) {
	if rbac.HasPermission(rbac.SystemAdmin, "nonexistent_permission") {
		t.Error("unknown permission should not be granted even to system_admin")
	}
}

func TestPermissionBoundaryDataSteward(t *testing.T) {
	// Data steward can do master_data_crud but not security_manage.
	if !rbac.HasPermission(rbac.DataSteward, "master_data_crud") {
		t.Error("data_steward should have master_data_crud")
	}
	if rbac.HasPermission(rbac.DataSteward, "security_manage") {
		t.Error("data_steward should NOT have security_manage")
	}
}

func TestPermissionBoundaryStandardUser(t *testing.T) {
	allowed := []string{"master_data_view", "playback_view", "reports_view"}
	for _, perm := range allowed {
		if !rbac.HasPermission(rbac.StandardUser, perm) {
			t.Errorf("standard_user should have %s", perm)
		}
	}

	denied := []string{"master_data_crud", "org_manage", "security_manage", "ingestion_manage"}
	for _, perm := range denied {
		if rbac.HasPermission(rbac.StandardUser, perm) {
			t.Errorf("standard_user should NOT have %s", perm)
		}
	}
}

func TestPermissionBoundaryOperationsAnalyst(t *testing.T) {
	if !rbac.HasPermission(rbac.OperationsAnalyst, "analytics_view") {
		t.Error("operations_analyst should have analytics_view")
	}
	if rbac.HasPermission(rbac.OperationsAnalyst, "master_data_crud") {
		t.Error("operations_analyst should NOT have master_data_crud")
	}
}
