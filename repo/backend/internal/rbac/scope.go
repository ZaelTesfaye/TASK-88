package rbac

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// CheckObjectScope verifies whether a user's scope covers the given city and department.
// Fail-closed: a non-admin user with no scope assigned is denied access.
// A wildcard value of "*" means unrestricted for that dimension.
func CheckObjectScope(user *models.User, objectCity, objectDept string) bool {
	// SystemAdmin has unrestricted scope
	if user.Role == SystemAdmin {
		return true
	}

	// Fail-closed: non-admin users with no scope at all are denied.
	if user.CityScope == "" && user.DepartmentScope == "" {
		return false
	}

	if user.CityScope != "" && user.CityScope != "*" && objectCity != "" {
		if user.CityScope != objectCity {
			return false
		}
	}

	if user.DepartmentScope != "" && user.DepartmentScope != "*" && objectDept != "" {
		if user.DepartmentScope != objectDept {
			return false
		}
	}

	return true
}

// ResolveScopeFilter applies the user's city and department scope to the given GORM query.
// It scopes based on the default column names "city" and "department".
func ResolveScopeFilter(user *models.User, db *gorm.DB) *gorm.DB {
	return EnforceScopeOnQuery(db, user, "city", "department")
}

// EnforceScopeOnQuery applies scope filtering to a GORM query using the specified
// column names for city and department.
// Fail-closed: non-admin users with no scope get a WHERE FALSE to return zero rows.
func EnforceScopeOnQuery(db *gorm.DB, user *models.User, cityCol, deptCol string) *gorm.DB {
	if user.Role == SystemAdmin {
		return db
	}

	// Fail-closed: no scope assigned → return nothing.
	if user.CityScope == "" && user.DepartmentScope == "" {
		return db.Where("1 = 0")
	}

	query := db

	if user.CityScope != "" && user.CityScope != "*" {
		query = query.Where(cityCol+" = ?", user.CityScope)
	}

	if user.DepartmentScope != "" && user.DepartmentScope != "*" {
		query = query.Where(deptCol+" = ?", user.DepartmentScope)
	}

	return query
}

// GetDescendantNodeIDs returns all descendant OrgNode IDs for a given parent node,
// traversing the tree recursively.
func GetDescendantNodeIDs(db *gorm.DB, nodeID uint) ([]uint, error) {
	var result []uint

	// Iterative breadth-first traversal
	queue := []uint{nodeID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		var children []models.OrgNode
		if err := db.Where("parent_id = ? AND is_active = ?", current, true).Find(&children).Error; err != nil {
			return nil, err
		}

		for _, child := range children {
			result = append(result, child.ID)
			queue = append(queue, child.ID)
		}
	}

	return result, nil
}
