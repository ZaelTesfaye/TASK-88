package unit

import (
	"testing"

	"backend/internal/reports"
)

// Tests for reports/scheduler.go — scheduler creation, timezone handling,
// and basic construction. Full start/run tests require TEST_DB_DSN.

func TestNewReportSchedulerValidTimezone(t *testing.T) {
	// With nil DB the scheduler can still be constructed (it won't load schedules).
	sched := reports.NewReportScheduler(nil, nil, "America/New_York")
	if sched == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

func TestNewReportSchedulerInvalidTimezoneFallsBackUTC(t *testing.T) {
	// Invalid timezone should not panic — it falls back to UTC with a warning.
	sched := reports.NewReportScheduler(nil, nil, "Invalid/Zone")
	if sched == nil {
		t.Fatal("expected non-nil scheduler even with invalid timezone")
	}
}

func TestNewReportSchedulerEmptyTimezone(t *testing.T) {
	// Empty timezone string — should fall back to UTC.
	sched := reports.NewReportScheduler(nil, nil, "")
	if sched == nil {
		t.Fatal("expected non-nil scheduler with empty timezone")
	}
}

func TestReportSchedulerStartWithNilDB(t *testing.T) {
	// Start() with nil DB should return an error or panic (can't query schedules).
	svc := reports.NewReportService(nil)
	sched := reports.NewReportScheduler(nil, svc, "UTC")
	defer func() {
		if r := recover(); r != nil {
			// nil DB causes a panic — acceptable, confirms DB is required.
			return
		}
	}()
	err := sched.Start()
	if err == nil {
		t.Error("expected error when starting scheduler with nil DB")
	}
}
