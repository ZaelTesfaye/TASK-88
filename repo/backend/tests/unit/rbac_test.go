package unit

import (
	"testing"

	"backend/internal/models"
	"backend/internal/rbac"
)

func TestSystemAdminHasAllPermissions(t *testing.T) {
	permissions := []string{
		"master_data_view",
		"master_data_crud",
		"master_data_import",
		"master_data_review",
		"version_draft",
		"version_activate",
		"analytics_view",
		"analytics_kpi",
		"reports_view",
		"reports_manage",
		"reports_download",
		"ingestion_view",
		"ingestion_manage",
		"playback_view",
		"audit_view",
		"audit_manage",
		"security_manage",
		"org_manage",
		"integration_manage",
	}

	for _, perm := range permissions {
		t.Run(perm, func(t *testing.T) {
			if !rbac.HasPermission(rbac.SystemAdmin, perm) {
				t.Errorf("system_admin should have permission %q", perm)
			}
		})
	}
}

func TestDataStewardPermissions(t *testing.T) {
	// Data steward CAN do master_data_crud.
	if !rbac.HasPermission(rbac.DataSteward, "master_data_crud") {
		t.Error("data_steward should have master_data_crud permission")
	}
	if !rbac.HasPermission(rbac.DataSteward, "master_data_import") {
		t.Error("data_steward should have master_data_import permission")
	}
	if !rbac.HasPermission(rbac.DataSteward, "master_data_review") {
		t.Error("data_steward should have master_data_review permission")
	}
	if !rbac.HasPermission(rbac.DataSteward, "version_draft") {
		t.Error("data_steward should have version_draft permission")
	}
	if !rbac.HasPermission(rbac.DataSteward, "playback_view") {
		t.Error("data_steward should have playback_view permission")
	}

	// Data steward CANNOT do analytics_view.
	if rbac.HasPermission(rbac.DataSteward, "analytics_view") {
		t.Error("data_steward should NOT have analytics_view permission")
	}
	if rbac.HasPermission(rbac.DataSteward, "org_manage") {
		t.Error("data_steward should NOT have org_manage permission")
	}
	if rbac.HasPermission(rbac.DataSteward, "security_manage") {
		t.Error("data_steward should NOT have security_manage permission")
	}
}

func TestOperationsAnalystPermissions(t *testing.T) {
	// Operations analyst CAN do analytics_view.
	if !rbac.HasPermission(rbac.OperationsAnalyst, "analytics_view") {
		t.Error("operations_analyst should have analytics_view permission")
	}
	if !rbac.HasPermission(rbac.OperationsAnalyst, "analytics_kpi") {
		t.Error("operations_analyst should have analytics_kpi permission")
	}
	if !rbac.HasPermission(rbac.OperationsAnalyst, "reports_manage") {
		t.Error("operations_analyst should have reports_manage permission")
	}
	if !rbac.HasPermission(rbac.OperationsAnalyst, "ingestion_view") {
		t.Error("operations_analyst should have ingestion_view permission")
	}
	if !rbac.HasPermission(rbac.OperationsAnalyst, "master_data_view") {
		t.Error("operations_analyst should have master_data_view permission")
	}

	// Operations analyst CANNOT do master_data_crud.
	if rbac.HasPermission(rbac.OperationsAnalyst, "master_data_crud") {
		t.Error("operations_analyst should NOT have master_data_crud permission")
	}
	if rbac.HasPermission(rbac.OperationsAnalyst, "org_manage") {
		t.Error("operations_analyst should NOT have org_manage permission")
	}
	if rbac.HasPermission(rbac.OperationsAnalyst, "security_manage") {
		t.Error("operations_analyst should NOT have security_manage permission")
	}
}

func TestStandardUserPermissions(t *testing.T) {
	// Standard user CAN view and playback.
	allowedPermissions := []string{
		"master_data_view",
		"playback_view",
		"reports_view",
	}
	for _, perm := range allowedPermissions {
		t.Run("allowed_"+perm, func(t *testing.T) {
			if !rbac.HasPermission(rbac.StandardUser, perm) {
				t.Errorf("standard_user should have permission %q", perm)
			}
		})
	}

	// Standard user CANNOT do admin-level operations.
	deniedPermissions := []string{
		"master_data_crud",
		"master_data_import",
		"analytics_view",
		"ingestion_manage",
		"org_manage",
		"security_manage",
		"audit_manage",
		"integration_manage",
	}
	for _, perm := range deniedPermissions {
		t.Run("denied_"+perm, func(t *testing.T) {
			if rbac.HasPermission(rbac.StandardUser, perm) {
				t.Errorf("standard_user should NOT have permission %q", perm)
			}
		})
	}
}

func TestScopeCheck(t *testing.T) {
	tests := []struct {
		name       string
		user       *models.User
		objectCity string
		objectDept string
		expected   bool
	}{
		{
			name: "matching city and department",
			user: &models.User{
				Role:            rbac.DataSteward,
				CityScope:       "NYC",
				DepartmentScope: "Finance",
			},
			objectCity: "NYC",
			objectDept: "Finance",
			expected:   true,
		},
		{
			name: "wrong city",
			user: &models.User{
				Role:            rbac.DataSteward,
				CityScope:       "NYC",
				DepartmentScope: "Finance",
			},
			objectCity: "LAX",
			objectDept: "Finance",
			expected:   false,
		},
		{
			name: "wrong department",
			user: &models.User{
				Role:            rbac.DataSteward,
				CityScope:       "NYC",
				DepartmentScope: "Finance",
			},
			objectCity: "NYC",
			objectDept: "Engineering",
			expected:   false,
		},
		{
			name: "system admin bypasses scope",
			user: &models.User{
				Role:            rbac.SystemAdmin,
				CityScope:       "NYC",
				DepartmentScope: "Finance",
			},
			objectCity: "LAX",
			objectDept: "Engineering",
			expected:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := rbac.CheckObjectScope(tc.user, tc.objectCity, tc.objectDept)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestScopeWildcard(t *testing.T) {
	tests := []struct {
		name       string
		user       *models.User
		objectCity string
		objectDept string
		expected   bool
	}{
		{
			name: "wildcard city scope allows any city",
			user: &models.User{
				Role:            rbac.DataSteward,
				CityScope:       "*",
				DepartmentScope: "Finance",
			},
			objectCity: "LAX",
			objectDept: "Finance",
			expected:   true,
		},
		{
			name: "wildcard department scope allows any department",
			user: &models.User{
				Role:            rbac.DataSteward,
				CityScope:       "NYC",
				DepartmentScope: "*",
			},
			objectCity: "NYC",
			objectDept: "Engineering",
			expected:   true,
		},
		{
			name: "both wildcards allow everything",
			user: &models.User{
				Role:            rbac.DataSteward,
				CityScope:       "*",
				DepartmentScope: "*",
			},
			objectCity: "LAX",
			objectDept: "Engineering",
			expected:   true,
		},
		{
			name: "empty scope (fail-closed) denies non-admin",
			user: &models.User{
				Role:            rbac.DataSteward,
				CityScope:       "",
				DepartmentScope: "",
			},
			objectCity: "LAX",
			objectDept: "Engineering",
			expected:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := rbac.CheckObjectScope(tc.user, tc.objectCity, tc.objectDept)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
