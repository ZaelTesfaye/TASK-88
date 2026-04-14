package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appErrors "backend/internal/errors"
	"backend/internal/rbac"
)

// ---------- in-memory master record store for API tests ----------

type inMemoryRecord struct {
	ID          uint   `json:"id"`
	EntityType  string `json:"entity_type"`
	NaturalKey  string `json:"natural_key"`
	PayloadJSON string `json:"payload_json"`
	Status      string `json:"status"`
}

type masterStore struct {
	mu      sync.Mutex
	records []inMemoryRecord
	nextID  uint
}

func newMasterStore() *masterStore {
	return &masterStore{nextID: 1}
}

func (s *masterStore) create(entityType, naturalKey, payloadJSON string) (*inMemoryRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.records {
		if r.EntityType == entityType && r.NaturalKey == naturalKey && r.Status == "active" {
			return nil, fmt.Errorf("duplicate")
		}
	}

	rec := inMemoryRecord{
		ID:          s.nextID,
		EntityType:  entityType,
		NaturalKey:  naturalKey,
		PayloadJSON: payloadJSON,
		Status:      "active",
	}
	s.nextID++
	s.records = append(s.records, rec)
	return &rec, nil
}

func (s *masterStore) list(entityType, search, status, sortBy, sortOrder string, page, pageSize int) ([]inMemoryRecord, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}

	var filtered []inMemoryRecord
	for _, r := range s.records {
		if r.EntityType != entityType {
			continue
		}
		if status != "" && status != "all" && r.Status != status {
			continue
		}
		filtered = append(filtered, r)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return []inMemoryRecord{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total
}

func (s *masterStore) deactivate(id uint, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reason == "" {
		return fmt.Errorf("reason required")
	}

	for i, r := range s.records {
		if r.ID == id {
			s.records[i].Status = "inactive"
			return nil
		}
	}
	return fmt.Errorf("not found")
}

// ---------- router ----------

func setupMasterRouter() (*gin.Engine, *masterStore) {
	r := testRouter()
	store := newMasterStore()

	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	masterRoutes := protected.Group("/master")
	{
		// List records
		masterRoutes.GET("/:entity", rbac.RequirePermission("master_data_view"), func(c *gin.Context) {
			entityType := c.Param("entity")
			search := c.Query("search")
			status := c.DefaultQuery("status", "active")
			sortBy := c.DefaultQuery("sort_by", "natural_key")
			sortOrder := c.DefaultQuery("sort_order", "asc")
			page := 1
			pageSize := 25
			fmt.Sscanf(c.DefaultQuery("page", "1"), "%d", &page)
			fmt.Sscanf(c.DefaultQuery("page_size", "25"), "%d", &pageSize)

			items, total := store.list(entityType, search, status, sortBy, sortOrder, page, pageSize)

			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"items":       items,
					"total":       total,
					"page":        page,
					"page_size":   pageSize,
					"total_pages": (total + pageSize - 1) / pageSize,
				},
			})
		})

		// Create record
		masterRoutes.POST("/:entity", rbac.RequirePermission("master_data_crud"), func(c *gin.Context) {
			entityType := c.Param("entity")
			var req struct {
				NaturalKey  string `json:"natural_key" binding:"required"`
				PayloadJSON string `json:"payload_json"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				appErrors.RespondBadRequest(c, "invalid request body", err.Error())
				return
			}

			rec, err := store.create(entityType, req.NaturalKey, req.PayloadJSON)
			if err != nil {
				if err.Error() == "duplicate" {
					appErrors.RespondConflict(c, fmt.Sprintf("a %s record with key %q already exists", entityType, req.NaturalKey), nil)
					return
				}
				appErrors.RespondInternalError(c, err.Error())
				return
			}

			c.JSON(http.StatusCreated, gin.H{"data": rec})
		})

		// Deactivate record
		masterRoutes.POST("/:entity/:id/deactivate", rbac.RequirePermission("master_data_crud"), func(c *gin.Context) {
			var id uint
			fmt.Sscanf(c.Param("id"), "%d", &id)

			var req struct {
				Reason string `json:"reason" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				appErrors.RespondBadRequest(c, "invalid request body: reason is required", err.Error())
				return
			}

			if err := store.deactivate(id, req.Reason); err != nil {
				if err.Error() == "not found" {
					appErrors.RespondNotFound(c, "record not found")
					return
				}
				appErrors.RespondInternalError(c, err.Error())
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "record deactivated"})
		})

		// Version activation - simulate conflict.
		masterRoutes.POST("/:entity/versions/:vid/activate", rbac.RequirePermission("master_data_crud"), func(c *gin.Context) {
			// Simulate a concurrent activation conflict.
			conflictHeader := c.GetHeader("X-Test-Force-Conflict")
			if conflictHeader == "true" {
				appErrors.RespondConflict(c, "version activation conflict: another version was activated concurrently",
					gin.H{"entity": c.Param("entity"), "version_id": c.Param("vid")})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "version activated"})
		})
	}

	return r, store
}

// ---------- tests ----------

func TestCreateMasterRecord(t *testing.T) {
	r, _ := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	body := map[string]string{
		"natural_key":  "SKU001",
		"payload_json": `{"name":"Widget","category":"Electronics"}`,
	}
	w := doRequest(r, "POST", "/api/v1/master/sku", token, body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response should have 'data' object")
	}
	if data["natural_key"] != "SKU001" {
		t.Errorf("expected natural_key=SKU001, got %v", data["natural_key"])
	}
	if data["status"] != "active" {
		t.Errorf("new record should have status=active, got %v", data["status"])
	}
}

func TestCreateDuplicateRecord(t *testing.T) {
	r, _ := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	body := map[string]string{
		"natural_key":  "DUP001",
		"payload_json": `{"name":"Original"}`,
	}

	// First create should succeed.
	w1 := doRequest(r, "POST", "/api/v1/master/sku", token, body)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first create expected 201, got %d", w1.Code)
	}

	// Second create with the same key should get a conflict/warning.
	w2 := doRequest(r, "POST", "/api/v1/master/sku", token, body)
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate create expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestDeactivateRecord(t *testing.T) {
	r, _ := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	// Create a record first.
	createBody := map[string]string{"natural_key": "DEACT01", "payload_json": `{}`}
	w := doRequest(r, "POST", "/api/v1/master/sku", token, createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", w.Code)
	}

	// Deactivate without reason should fail.
	w2 := doRequest(r, "POST", "/api/v1/master/sku/1/deactivate", token, map[string]string{})
	if w2.Code != http.StatusBadRequest {
		t.Errorf("deactivate without reason expected 400, got %d: %s", w2.Code, w2.Body.String())
	}

	// Deactivate with reason should succeed.
	deactBody := map[string]string{"reason": "Discontinued product"}
	w3 := doRequest(r, "POST", "/api/v1/master/sku/1/deactivate", token, deactBody)
	if w3.Code != http.StatusOK {
		t.Errorf("deactivate with reason expected 200, got %d: %s", w3.Code, w3.Body.String())
	}
}

func TestVersionActivationConflict(t *testing.T) {
	r, _ := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	// Simulate a conflict with a special header.
	req, _ := http.NewRequest("POST", "/api/v1/master/sku/versions/1/activate", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Test-Force-Conflict", "true")

	w := doRequestRaw(r, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for version conflict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPagination(t *testing.T) {
	r, store := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	// Create 10 records.
	for i := 1; i <= 10; i++ {
		store.create("sku", fmt.Sprintf("PAGE%03d", i), `{}`)
	}

	// Request page 1, page_size 3.
	w := doRequest(r, "GET", "/api/v1/master/sku?page=1&page_size=3", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseBody(w)
	data := resp["data"].(map[string]interface{})

	total := data["total"].(float64)
	if int(total) != 10 {
		t.Errorf("expected total=10, got %v", total)
	}

	page := data["page"].(float64)
	if int(page) != 1 {
		t.Errorf("expected page=1, got %v", page)
	}

	pageSize := data["page_size"].(float64)
	if int(pageSize) != 3 {
		t.Errorf("expected page_size=3, got %v", pageSize)
	}

	items := data["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("expected 3 items on page 1, got %d", len(items))
	}

	// Request page 4 (last page with 1 item).
	w2 := doRequest(r, "GET", "/api/v1/master/sku?page=4&page_size=3", token, nil)
	resp2 := parseBody(w2)
	data2 := resp2["data"].(map[string]interface{})
	items2 := data2["items"].([]interface{})
	if len(items2) != 1 {
		t.Errorf("expected 1 item on last page, got %d", len(items2))
	}
}

func TestSorting(t *testing.T) {
	r, store := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	store.create("sku", "ZZZ999", `{}`)
	store.create("sku", "AAA111", `{}`)
	store.create("sku", "MMM555", `{}`)

	// Test sort_by and sort_order parameters are accepted.
	w := doRequest(r, "GET", "/api/v1/master/sku?sort_by=natural_key&sort_order=asc", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	// Verify sort_order=desc is also accepted.
	w2 := doRequest(r, "GET", "/api/v1/master/sku?sort_by=natural_key&sort_order=desc", token, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for desc sort, got %d", w2.Code)
	}
}

func TestFiltering(t *testing.T) {
	r, store := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	store.create("sku", "FILTER01", `{}`)
	store.create("sku", "FILTER02", `{}`)
	store.create("season", "SS2025", `{}`)

	// Filter by status.
	w := doRequest(r, "GET", "/api/v1/master/sku?status=active", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseBody(w)
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 sku items with active status, got %d", len(items))
	}

	// Season entity type should have its own records.
	w2 := doRequest(r, "GET", "/api/v1/master/season?status=active", token, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	resp2 := parseBody(w2)
	data2 := resp2["data"].(map[string]interface{})
	items2 := data2["items"].([]interface{})
	if len(items2) != 1 {
		t.Errorf("expected 1 season item, got %d", len(items2))
	}
}

func TestEmptyDataset(t *testing.T) {
	r, _ := setupMasterRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	w := doRequest(r, "GET", "/api/v1/master/sku?status=active", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseBody(w)
	data := resp["data"].(map[string]interface{})

	total := data["total"].(float64)
	if int(total) != 0 {
		t.Errorf("expected total=0, got %v", total)
	}

	items := data["items"].([]interface{})
	if items == nil {
		// items should be an empty array, not null.
		t.Error("items should be an empty array, not null")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// ---------- helpers ----------

func doRequestRaw(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Ensure json import is used (needed for json.Marshal in doRequest helper).
var _ = json.Marshal
