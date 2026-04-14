package api

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"backend/internal/auth"
	appErrors "backend/internal/errors"
	"backend/internal/rbac"
)

// ---------- in-memory audit store ----------

type inMemoryAuditLog struct {
	ID            uint   `json:"id"`
	ActorUserID   *uint  `json:"actor_user_id"`
	ActionType    string `json:"action_type"`
	TargetType    string `json:"target_type"`
	TargetID      string `json:"target_id"`
	SensitiveRead bool   `json:"sensitive_read"`
}

type inMemoryDeleteRequest struct {
	ID          uint   `json:"id"`
	RequestedBy uint   `json:"requested_by"`
	Reason      string `json:"reason"`
	State       string `json:"state"`
	ApproverOne *uint  `json:"approver_one"`
	ApproverTwo *uint  `json:"approver_two"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
}

type auditStore struct {
	mu             sync.Mutex
	logs           []inMemoryAuditLog
	deleteRequests []inMemoryDeleteRequest
	nextLogID      uint
	nextReqID      uint
}

func newAuditStore() *auditStore {
	return &auditStore{nextLogID: 1, nextReqID: 1}
}

func (s *auditStore) addLog(actorID *uint, actionType, targetType, targetID string, sensitiveRead bool) *inMemoryAuditLog {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := inMemoryAuditLog{
		ID:            s.nextLogID,
		ActorUserID:   actorID,
		ActionType:    actionType,
		TargetType:    targetType,
		TargetID:      targetID,
		SensitiveRead: sensitiveRead,
	}
	s.nextLogID++
	s.logs = append(s.logs, log)
	return &log
}

func (s *auditStore) createDeleteRequest(requestedBy uint, reason, targetType, targetID string) *inMemoryDeleteRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	req := inMemoryDeleteRequest{
		ID:          s.nextReqID,
		RequestedBy: requestedBy,
		Reason:      reason,
		State:       "pending",
		TargetType:  targetType,
		TargetID:    targetID,
	}
	s.nextReqID++
	s.deleteRequests = append(s.deleteRequests, req)
	return &req
}

func (s *auditStore) getDeleteRequest(id uint) *inMemoryDeleteRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.deleteRequests {
		if s.deleteRequests[i].ID == id {
			return &s.deleteRequests[i]
		}
	}
	return nil
}

func (s *auditStore) approve(id uint, approverID uint) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.deleteRequests {
		req := &s.deleteRequests[i]
		if req.ID != id {
			continue
		}

		if req.State != "pending" && req.State != "partially_approved" {
			return "", fmt.Errorf("cannot approve in state %s", req.State)
		}

		// Requester cannot approve their own request.
		if req.RequestedBy == approverID {
			return "", fmt.Errorf("requester cannot approve")
		}

		if req.ApproverOne == nil {
			req.ApproverOne = &approverID
			req.State = "partially_approved"
			return "first approval recorded", nil
		}

		// Second approval: different from first.
		if *req.ApproverOne == approverID {
			return "", fmt.Errorf("same user cannot approve twice")
		}

		req.ApproverTwo = &approverID
		req.State = "approved"
		return "fully approved", nil
	}
	return "", fmt.Errorf("not found")
}

func (s *auditStore) execute(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.deleteRequests {
		if s.deleteRequests[i].ID == id {
			if s.deleteRequests[i].State != "approved" {
				return fmt.Errorf("must be approved first")
			}
			s.deleteRequests[i].State = "executed"
			return nil
		}
	}
	return fmt.Errorf("not found")
}

// ---------- router ----------

func setupAuditRouter() (*gin.Engine, *auditStore) {
	r := testRouter()
	store := newAuditStore()

	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	auditRoutes := protected.Group("/audit")
	auditRoutes.Use(rbac.RequireRole(rbac.SystemAdmin))
	{
		// Simulate a mutation that creates an audit entry.
		auditRoutes.POST("/test-mutation", func(c *gin.Context) {
			user := auth.GetCurrentUser(c)
			if user == nil {
				appErrors.RespondUnauthorized(c, "auth required")
				return
			}
			store.addLog(&user.ID, "CREATE", "test_entity", "42", false)
			c.JSON(http.StatusOK, gin.H{"message": "mutation completed"})
		})

		// Simulate a sensitive field read.
		auditRoutes.GET("/sensitive-read", func(c *gin.Context) {
			user := auth.GetCurrentUser(c)
			if user == nil {
				appErrors.RespondUnauthorized(c, "auth required")
				return
			}
			store.addLog(&user.ID, "SENSITIVE_READ", "user", "1", true)
			c.JSON(http.StatusOK, gin.H{"message": "sensitive data read"})
		})

		// List logs.
		auditRoutes.GET("/logs", func(c *gin.Context) {
			store.mu.Lock()
			logs := make([]inMemoryAuditLog, len(store.logs))
			copy(logs, store.logs)
			store.mu.Unlock()

			c.JSON(http.StatusOK, gin.H{"data": logs, "total": len(logs)})
		})

		// Create delete request.
		auditRoutes.POST("/delete-requests", func(c *gin.Context) {
			user := auth.GetCurrentUser(c)
			if user == nil {
				appErrors.RespondUnauthorized(c, "auth required")
				return
			}

			var body struct {
				Reason     string `json:"reason" binding:"required"`
				TargetType string `json:"target_type" binding:"required"`
				TargetID   string `json:"target_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				appErrors.RespondBadRequest(c, "invalid body", err.Error())
				return
			}

			req := store.createDeleteRequest(user.ID, body.Reason, body.TargetType, body.TargetID)
			c.JSON(http.StatusCreated, req)
		})

		// Approve delete request.
		auditRoutes.POST("/delete-requests/:id/approve", func(c *gin.Context) {
			user := auth.GetCurrentUser(c)
			if user == nil {
				appErrors.RespondUnauthorized(c, "auth required")
				return
			}

			var id uint
			fmt.Sscanf(c.Param("id"), "%d", &id)

			msg, err := store.approve(id, user.ID)
			if err != nil {
				if err.Error() == "requester cannot approve" || err.Error() == "same user cannot approve twice" {
					appErrors.RespondForbidden(c, err.Error())
					return
				}
				appErrors.RespondConflict(c, err.Error(), nil)
				return
			}

			req := store.getDeleteRequest(id)
			c.JSON(http.StatusOK, gin.H{"message": msg, "data": req})
		})

		// Execute delete request.
		auditRoutes.POST("/delete-requests/:id/execute", func(c *gin.Context) {
			var id uint
			fmt.Sscanf(c.Param("id"), "%d", &id)

			if err := store.execute(id); err != nil {
				appErrors.RespondConflict(c, err.Error(), nil)
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})
	}

	return r, store
}

// ---------- tests ----------

func TestAuditLogCreated(t *testing.T) {
	r, store := setupAuditRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	// Perform a mutation.
	w := doRequest(r, "POST", "/api/v1/audit/test-mutation", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("mutation expected 200, got %d", w.Code)
	}

	// Check that an audit log was created.
	store.mu.Lock()
	logCount := len(store.logs)
	store.mu.Unlock()

	if logCount != 1 {
		t.Fatalf("expected 1 audit log, got %d", logCount)
	}

	log := store.logs[0]
	if log.ActionType != "CREATE" {
		t.Errorf("expected action CREATE, got %s", log.ActionType)
	}
	if log.TargetType != "test_entity" {
		t.Errorf("expected target_type test_entity, got %s", log.TargetType)
	}
	if log.TargetID != "42" {
		t.Errorf("expected target_id 42, got %s", log.TargetID)
	}
	if log.ActorUserID == nil || *log.ActorUserID != 1 {
		t.Error("expected actor_user_id to be 1")
	}
}

func TestSensitiveReadLogged(t *testing.T) {
	r, store := setupAuditRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	w := doRequest(r, "GET", "/api/v1/audit/sensitive-read", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	found := false
	for _, log := range store.logs {
		if log.ActionType == "SENSITIVE_READ" && log.SensitiveRead {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a SENSITIVE_READ audit log with sensitive_read=true")
	}
}

func TestDualApprovalRequired(t *testing.T) {
	r, _ := setupAuditRouter()

	// User 10 creates the request.
	token10 := signToken(10, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	createBody := map[string]string{
		"reason":      "Clean up old data",
		"target_type": "audit_log",
		"target_id":   "123",
	}
	w := doRequest(r, "POST", "/api/v1/audit/delete-requests", token10, createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// User 10 tries to approve their own request -> should fail.
	w2 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token10, nil)
	if w2.Code != http.StatusForbidden {
		t.Errorf("self-approval expected 403, got %d: %s", w2.Code, w2.Body.String())
	}

	// User 20 provides first approval.
	token20 := signToken(20, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	w3 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token20, nil)
	if w3.Code != http.StatusOK {
		t.Fatalf("first approval expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	// User 20 tries to approve again (same user, second approval) -> should fail.
	w4 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token20, nil)
	if w4.Code != http.StatusForbidden {
		t.Errorf("duplicate approval expected 403, got %d: %s", w4.Code, w4.Body.String())
	}

	// User 30 provides second approval -> succeeds.
	token30 := signToken(30, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	w5 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token30, nil)
	if w5.Code != http.StatusOK {
		t.Fatalf("second approval expected 200, got %d: %s", w5.Code, w5.Body.String())
	}

	resp := parseBody(w5)
	msg, _ := resp["message"].(string)
	if msg != "fully approved" {
		t.Errorf("expected 'fully approved', got %q", msg)
	}
}

func TestDeleteRequestWorkflow(t *testing.T) {
	r, _ := setupAuditRouter()

	// Step 1: Create delete request (user 10).
	token10 := signToken(10, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	createBody := map[string]string{
		"reason":      "GDPR compliance",
		"target_type": "user_data",
		"target_id":   "456",
	}
	w1 := doRequest(r, "POST", "/api/v1/audit/delete-requests", token10, createBody)
	if w1.Code != http.StatusCreated {
		t.Fatalf("step 1 create: expected 201, got %d", w1.Code)
	}
	resp1 := parseBody(w1)
	state1 := resp1["state"].(string)
	if state1 != "pending" {
		t.Errorf("step 1: expected state=pending, got %s", state1)
	}

	// Step 2: First approval (user 20).
	token20 := signToken(20, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	w2 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token20, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("step 2 first approve: expected 200, got %d", w2.Code)
	}

	// Step 3: Second approval (user 30).
	token30 := signToken(30, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	w3 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token30, nil)
	if w3.Code != http.StatusOK {
		t.Fatalf("step 3 second approve: expected 200, got %d", w3.Code)
	}

	// Step 4: Execute the approved deletion.
	w4 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/execute", token10, nil)
	if w4.Code != http.StatusOK {
		t.Fatalf("step 4 execute: expected 200, got %d: %s", w4.Code, w4.Body.String())
	}

	// Trying to execute again should fail (already executed).
	w5 := doRequest(r, "POST", "/api/v1/audit/delete-requests/1/execute", token10, nil)
	if w5.Code == http.StatusOK {
		t.Error("re-executing should fail")
	}
}
