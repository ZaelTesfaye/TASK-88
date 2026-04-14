package org

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	appErrors "backend/internal/errors"
	"backend/internal/models"
)

// OrgTreeNode is the nested tree response.
type OrgTreeNode struct {
	models.OrgNode
	Children []OrgTreeNode `json:"children"`
}

// CreateNodeRequest holds the data needed to create an org node.
type CreateNodeRequest struct {
	ParentID   *uint  `json:"parent_id"`
	LevelCode  string `json:"level_code" binding:"required"`
	LevelLabel string `json:"level_label" binding:"required"`
	Name       string `json:"name" binding:"required"`
	City       string `json:"city"`
	Department string `json:"department"`
	SortOrder  int    `json:"sort_order"`
}

// UpdateNodeRequest holds the data needed to update an org node.
type UpdateNodeRequest struct {
	ParentID   *uint   `json:"parent_id"`
	LevelCode  *string `json:"level_code"`
	LevelLabel *string `json:"level_label"`
	Name       *string `json:"name"`
	City       *string `json:"city"`
	Department *string `json:"department"`
	SortOrder  *int    `json:"sort_order"`
	IsActive   *bool   `json:"is_active"`
}

// SwitchContextRequest holds the data for switching user context.
type SwitchContextRequest struct {
	NodeID uint `json:"node_id" binding:"required"`
}

// OrgService provides org tree management operations.
type OrgService struct {
	db *gorm.DB
}

// NewOrgService creates a new OrgService.
func NewOrgService(db *gorm.DB) *OrgService {
	return &OrgService{db: db}
}

// validLevelCodes lists the acceptable level codes.
var validLevelCodes = map[string]bool{
	"company":    true,
	"region":     true,
	"city":       true,
	"department": true,
	"team":       true,
	"unit":       true,
}

// GetTree returns the full org tree as a nested structure.
func (s *OrgService) GetTree() ([]OrgTreeNode, error) {
	var nodes []models.OrgNode
	if err := s.db.Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&nodes).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load org nodes: %v", err))
	}

	return buildTree(nodes), nil
}

// buildTree builds a nested tree from a flat list of nodes.
func buildTree(nodes []models.OrgNode) []OrgTreeNode {
	nodeMap := make(map[uint]*OrgTreeNode, len(nodes))
	var roots []OrgTreeNode

	// First pass: create all tree nodes.
	for _, n := range nodes {
		tn := OrgTreeNode{
			OrgNode:  n,
			Children: []OrgTreeNode{},
		}
		nodeMap[n.ID] = &tn
	}

	// Second pass: attach children to parents.
	for _, n := range nodes {
		tn := nodeMap[n.ID]
		if n.ParentID == nil {
			roots = append(roots, *tn)
		} else {
			parent, ok := nodeMap[*n.ParentID]
			if ok {
				parent.Children = append(parent.Children, *tn)
			} else {
				// Orphan node - treat as root.
				roots = append(roots, *tn)
			}
		}
	}

	if roots == nil {
		roots = []OrgTreeNode{}
	}

	// Rebuild the tree to ensure nested children are correctly populated.
	return rebuildTreeFromMap(nodeMap, roots, nodes)
}

// rebuildTreeFromMap performs a recursive rebuild so children of children are correct.
func rebuildTreeFromMap(nodeMap map[uint]*OrgTreeNode, _ []OrgTreeNode, allNodes []models.OrgNode) []OrgTreeNode {
	childrenOf := make(map[uint][]uint)
	var rootIDs []uint

	for _, n := range allNodes {
		if n.ParentID == nil {
			rootIDs = append(rootIDs, n.ID)
		} else {
			childrenOf[*n.ParentID] = append(childrenOf[*n.ParentID], n.ID)
		}
	}

	var buildSubtree func(id uint) OrgTreeNode
	buildSubtree = func(id uint) OrgTreeNode {
		tn := *nodeMap[id]
		tn.Children = []OrgTreeNode{}
		for _, childID := range childrenOf[id] {
			tn.Children = append(tn.Children, buildSubtree(childID))
		}
		return tn
	}

	result := make([]OrgTreeNode, 0, len(rootIDs))
	for _, rid := range rootIDs {
		result = append(result, buildSubtree(rid))
	}
	if result == nil {
		result = []OrgTreeNode{}
	}
	return result
}

// CreateNode creates a new org node with validation.
func (s *OrgService) CreateNode(req CreateNodeRequest) (*models.OrgNode, error) {
	// Validate level_code.
	if err := validateLevelCode(req.LevelCode); err != nil {
		return nil, err
	}

	// Validate level_label.
	if strings.TrimSpace(req.LevelLabel) == "" {
		return nil, appErrors.ValidationError("level_label is required", nil)
	}

	// Validate name.
	if strings.TrimSpace(req.Name) == "" {
		return nil, appErrors.ValidationError("name is required", nil)
	}

	// Validate parent exists if parent_id provided.
	if req.ParentID != nil {
		var parent models.OrgNode
		if err := s.db.First(&parent, *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, appErrors.NotFound(fmt.Sprintf("parent node %d not found", *req.ParentID))
			}
			return nil, appErrors.InternalError(fmt.Sprintf("failed to look up parent: %v", err))
		}
	}

	// Enforce unique name within same parent scope.
	if err := s.checkUniqueName(0, req.ParentID, req.Name); err != nil {
		return nil, err
	}

	node := models.OrgNode{
		ParentID:   req.ParentID,
		LevelCode:  req.LevelCode,
		LevelLabel: req.LevelLabel,
		Name:       strings.TrimSpace(req.Name),
		City:       strings.TrimSpace(req.City),
		Department: strings.TrimSpace(req.Department),
		SortOrder:  req.SortOrder,
		IsActive:   true,
	}

	if err := s.db.Create(&node).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to create org node: %v", err))
	}

	return &node, nil
}

// UpdateNode updates an existing org node.
func (s *OrgService) UpdateNode(id uint, req UpdateNodeRequest) (*models.OrgNode, error) {
	var node models.OrgNode
	if err := s.db.First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, appErrors.NotFound(fmt.Sprintf("org node %d not found", id))
		}
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load org node: %v", err))
	}

	// Apply fields if provided.
	if req.LevelCode != nil {
		if err := validateLevelCode(*req.LevelCode); err != nil {
			return nil, err
		}
		node.LevelCode = *req.LevelCode
	}
	if req.LevelLabel != nil {
		if strings.TrimSpace(*req.LevelLabel) == "" {
			return nil, appErrors.ValidationError("level_label cannot be empty", nil)
		}
		node.LevelLabel = *req.LevelLabel
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, appErrors.ValidationError("name cannot be empty", nil)
		}
		node.Name = strings.TrimSpace(*req.Name)
	}
	if req.City != nil {
		node.City = strings.TrimSpace(*req.City)
	}
	if req.Department != nil {
		node.Department = strings.TrimSpace(*req.Department)
	}
	if req.SortOrder != nil {
		node.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		node.IsActive = *req.IsActive
	}

	// Handle parent change.
	if req.ParentID != nil {
		newParentID := *req.ParentID

		// Cannot be own parent.
		if newParentID == id {
			return nil, appErrors.ValidationError("a node cannot be its own parent", nil)
		}

		// Validate new parent exists.
		var parent models.OrgNode
		if err := s.db.First(&parent, newParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, appErrors.NotFound(fmt.Sprintf("parent node %d not found", newParentID))
			}
			return nil, appErrors.InternalError(fmt.Sprintf("failed to look up parent: %v", err))
		}

		// Anti-cycle check: new parent cannot be a descendant of this node.
		if err := s.ValidateNoCycle(id, newParentID); err != nil {
			return nil, err
		}

		node.ParentID = &newParentID
	}

	// Check unique name within same parent scope (exclude self).
	nameToCheck := node.Name
	if req.Name != nil {
		nameToCheck = strings.TrimSpace(*req.Name)
	}
	if err := s.checkUniqueName(id, node.ParentID, nameToCheck); err != nil {
		return nil, err
	}

	if err := s.db.Save(&node).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to update org node: %v", err))
	}

	return &node, nil
}

// DeleteNode deletes a node after validating it has no active children.
// Cascade deletes context assignments.
func (s *OrgService) DeleteNode(id uint) error {
	var node models.OrgNode
	if err := s.db.First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErrors.NotFound(fmt.Sprintf("org node %d not found", id))
		}
		return appErrors.InternalError(fmt.Sprintf("failed to load org node: %v", err))
	}

	// Check for active children.
	var childCount int64
	if err := s.db.Model(&models.OrgNode{}).Where("parent_id = ? AND is_active = ?", id, true).Count(&childCount).Error; err != nil {
		return appErrors.InternalError(fmt.Sprintf("failed to count children: %v", err))
	}
	if childCount > 0 {
		return appErrors.ValidationError(
			fmt.Sprintf("cannot delete node %d: it has %d active child node(s); reassign or delete them first", id, childCount),
			map[string]interface{}{"active_children": childCount},
		)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete context assignments for this node.
		if err := tx.Where("org_node_id = ?", id).Delete(&models.ContextAssignment{}).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to delete context assignments: %v", err))
		}

		// Delete the node itself.
		if err := tx.Delete(&models.OrgNode{}, id).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to delete org node: %v", err))
		}

		return nil
	})
}

// GetDescendants returns all descendant node IDs (recursive, breadth-first).
func (s *OrgService) GetDescendants(nodeID uint) ([]uint, error) {
	var result []uint
	queue := []uint{nodeID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		var children []models.OrgNode
		if err := s.db.Where("parent_id = ? AND is_active = ?", current, true).Find(&children).Error; err != nil {
			return nil, appErrors.InternalError(fmt.Sprintf("failed to load descendants: %v", err))
		}

		for _, child := range children {
			result = append(result, child.ID)
			queue = append(queue, child.ID)
		}
	}

	return result, nil
}

// ValidateNoCycle checks if setting parentID for nodeID would create a cycle.
// It walks up the parent chain from parentID to see if nodeID is encountered.
func (s *OrgService) ValidateNoCycle(nodeID, parentID uint) error {
	if nodeID == parentID {
		return appErrors.ValidationError("a node cannot be its own parent", nil)
	}

	// Walk up the parent chain from parentID.
	visited := make(map[uint]bool)
	current := parentID

	for {
		if visited[current] {
			// Unexpected existing cycle in the data.
			return appErrors.ValidationError("cycle detected in org tree", nil)
		}
		visited[current] = true

		var node models.OrgNode
		if err := s.db.Select("id, parent_id").First(&node, current).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Reached a node that doesn't exist - no cycle.
				return nil
			}
			return appErrors.InternalError(fmt.Sprintf("failed to traverse parent chain: %v", err))
		}

		if node.ParentID == nil {
			// Reached root, no cycle.
			return nil
		}

		if *node.ParentID == nodeID {
			return appErrors.ValidationError(
				fmt.Sprintf("setting parent to %d would create a cycle: node %d is a descendant of node %d", parentID, parentID, nodeID),
				nil,
			)
		}

		current = *node.ParentID
	}
}

// SwitchContext sets the user's current org context.
func (s *OrgService) SwitchContext(userID, nodeID uint) error {
	// Validate node exists and is active.
	var node models.OrgNode
	if err := s.db.Where("id = ? AND is_active = ?", nodeID, true).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErrors.NotFound(fmt.Sprintf("org node %d not found or inactive", nodeID))
		}
		return appErrors.InternalError(fmt.Sprintf("failed to look up org node: %v", err))
	}

	// Validate user exists.
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErrors.NotFound(fmt.Sprintf("user %d not found", userID))
		}
		return appErrors.InternalError(fmt.Sprintf("failed to look up user: %v", err))
	}

	// Upsert context assignment.
	var existing models.ContextAssignment
	err := s.db.Where("user_id = ?", userID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		assignment := models.ContextAssignment{
			UserID:    userID,
			OrgNodeID: nodeID,
		}
		if err := s.db.Create(&assignment).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to create context assignment: %v", err))
		}
		return nil
	}
	if err != nil {
		return appErrors.InternalError(fmt.Sprintf("failed to look up context assignment: %v", err))
	}

	existing.OrgNodeID = nodeID
	if err := s.db.Save(&existing).Error; err != nil {
		return appErrors.InternalError(fmt.Sprintf("failed to update context assignment: %v", err))
	}

	return nil
}

// GetUserContext returns the current context node for a user and all descendant IDs for scope filtering.
func (s *OrgService) GetUserContext(userID uint) (*models.OrgNode, []uint, error) {
	var assignment models.ContextAssignment
	if err := s.db.Where("user_id = ?", userID).First(&assignment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil
		}
		return nil, nil, appErrors.InternalError(fmt.Sprintf("failed to load context assignment: %v", err))
	}

	var node models.OrgNode
	if err := s.db.First(&node, assignment.OrgNodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, appErrors.NotFound("context node no longer exists")
		}
		return nil, nil, appErrors.InternalError(fmt.Sprintf("failed to load context node: %v", err))
	}

	descendants, err := s.GetDescendants(node.ID)
	if err != nil {
		return nil, nil, err
	}

	// Include the node itself in the scope.
	scopeIDs := append([]uint{node.ID}, descendants...)

	return &node, scopeIDs, nil
}

// GetBreadcrumb returns the path from root to the given node.
func (s *OrgService) GetBreadcrumb(nodeID uint) ([]models.OrgNode, error) {
	var breadcrumb []models.OrgNode
	current := nodeID
	visited := make(map[uint]bool)

	for {
		if visited[current] {
			break
		}
		visited[current] = true

		var node models.OrgNode
		if err := s.db.First(&node, current).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return nil, appErrors.InternalError(fmt.Sprintf("failed to load breadcrumb node: %v", err))
		}

		breadcrumb = append([]models.OrgNode{node}, breadcrumb...)

		if node.ParentID == nil {
			break
		}
		current = *node.ParentID
	}

	return breadcrumb, nil
}

// checkUniqueName checks that no sibling node (under the same parent) has the same name.
// excludeID is used when updating to exclude the node being updated.
func (s *OrgService) checkUniqueName(excludeID uint, parentID *uint, name string) *appErrors.AppError {
	query := s.db.Model(&models.OrgNode{}).Where("LOWER(name) = LOWER(?) AND is_active = ?", strings.TrimSpace(name), true)

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return appErrors.InternalError(fmt.Sprintf("failed to check name uniqueness: %v", err))
	}
	if count > 0 {
		return appErrors.Conflict(
			fmt.Sprintf("a node named %q already exists under the same parent", name),
			map[string]interface{}{"name": name},
		)
	}
	return nil
}

// validateLevelCode checks that the level code is valid.
func validateLevelCode(code string) *appErrors.AppError {
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" {
		return appErrors.ValidationError("level_code is required", nil)
	}
	if !validLevelCodes[code] {
		return appErrors.ValidationError(
			fmt.Sprintf("invalid level_code %q; valid values are: company, region, city, department, team, unit", code),
			map[string]interface{}{"valid_values": []string{"company", "region", "city", "department", "team", "unit"}},
		)
	}
	return nil
}
