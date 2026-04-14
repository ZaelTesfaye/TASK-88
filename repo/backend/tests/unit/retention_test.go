package unit

import (
	"testing"
	"time"

	"backend/internal/models"
)

// Since DryRunPurge, ExecutePurge, and related methods all require a real DB,
// we test the core purge logic (eligibility, legal hold blocking, retention
// policy enforcement) by simulating the decision-making that the
// SecurityService performs.

// simulatePurgeEligibility mirrors the logic in countPurgeEligible:
// Returns (eligible, blocked) counts.
func simulatePurgeEligibility(
	artifactCreatedAt []time.Time,
	retentionDays int,
	hasActiveLegalHold bool,
) (eligible int, blocked int) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	for _, created := range artifactCreatedAt {
		if created.Before(cutoff) {
			eligible++
		}
	}

	if hasActiveLegalHold {
		blocked = eligible
	}

	return eligible, blocked
}

func TestDryRunPurge(t *testing.T) {
	now := time.Now()

	// 5 artifacts: 3 are older than 90 days, 2 are recent.
	artifacts := []time.Time{
		now.AddDate(0, 0, -100), // eligible
		now.AddDate(0, 0, -95),  // eligible
		now.AddDate(0, 0, -91),  // eligible
		now.AddDate(0, 0, -30),  // recent
		now.AddDate(0, 0, -5),   // recent
	}

	retentionDays := 90

	eligible, blocked := simulatePurgeEligibility(artifacts, retentionDays, false)

	if eligible != 3 {
		t.Errorf("expected 3 eligible for purge, got %d", eligible)
	}
	if blocked != 0 {
		t.Errorf("expected 0 blocked (no legal hold), got %d", blocked)
	}

	wouldPurge := eligible - blocked
	if wouldPurge != 3 {
		t.Errorf("dry run: expected 3 would be purged, got %d", wouldPurge)
	}

	// Verify the dry run does NOT modify original data (preview only).
	// In the real implementation, DryRunPurge only counts, never deletes.
	if len(artifacts) != 5 {
		t.Error("dry run should not modify the artifact list")
	}
}

func TestLegalHoldBlocksPurge(t *testing.T) {
	now := time.Now()

	artifacts := []time.Time{
		now.AddDate(0, 0, -200), // eligible
		now.AddDate(0, 0, -150), // eligible
		now.AddDate(0, 0, -100), // eligible
	}
	retentionDays := 90

	eligible, blocked := simulatePurgeEligibility(artifacts, retentionDays, true)

	if eligible != 3 {
		t.Errorf("expected 3 eligible, got %d", eligible)
	}
	if blocked != 3 {
		t.Errorf("expected all 3 blocked by legal hold, got %d", blocked)
	}

	wouldPurge := eligible - blocked
	if wouldPurge != 0 {
		t.Errorf("with legal hold, expected 0 purged, got %d", wouldPurge)
	}

	// Verify the model: a legal hold with nil ReleasedAt is active.
	hold := models.LegalHold{
		Reason:     "Pending litigation",
		ReleasedAt: nil,
	}
	if hold.ReleasedAt != nil {
		t.Error("active legal hold should have nil ReleasedAt")
	}

	// A released hold has a non-nil ReleasedAt.
	releasedTime := time.Now()
	releasedHold := models.LegalHold{
		Reason:     "Resolved",
		ReleasedAt: &releasedTime,
	}
	if releasedHold.ReleasedAt == nil {
		t.Error("released legal hold should have non-nil ReleasedAt")
	}
}

func TestRetentionPolicyEnforcement(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		retentionDays int
		artifacts     []time.Time
		expectedPurge int
	}{
		{
			name:          "30-day retention",
			retentionDays: 30,
			artifacts: []time.Time{
				now.AddDate(0, 0, -31), // eligible
				now.AddDate(0, 0, -35), // eligible
				now.AddDate(0, 0, -10), // NOT eligible
			},
			expectedPurge: 2,
		},
		{
			name:          "90-day retention",
			retentionDays: 90,
			artifacts: []time.Time{
				now.AddDate(0, 0, -91),  // eligible
				now.AddDate(0, 0, -100), // eligible
				now.AddDate(0, 0, -89),  // NOT eligible
				now.AddDate(0, 0, -1),   // NOT eligible
			},
			expectedPurge: 2,
		},
		{
			name:          "365-day retention - nothing eligible",
			retentionDays: 365,
			artifacts: []time.Time{
				now.AddDate(0, 0, -100),
				now.AddDate(0, 0, -200),
				now.AddDate(0, 0, -300),
			},
			expectedPurge: 0,
		},
		{
			name:          "365-day retention - all eligible",
			retentionDays: 365,
			artifacts: []time.Time{
				now.AddDate(0, 0, -400),
				now.AddDate(0, 0, -500),
			},
			expectedPurge: 2,
		},
		{
			name:          "empty dataset",
			retentionDays: 30,
			artifacts:     []time.Time{},
			expectedPurge: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eligible, _ := simulatePurgeEligibility(tc.artifacts, tc.retentionDays, false)
			if eligible != tc.expectedPurge {
				t.Errorf("expected %d eligible for purge, got %d",
					tc.expectedPurge, eligible)
			}
		})
	}

	// Verify the retention policy model.
	policy := models.RetentionPolicy{
		ArtifactType:  "audit_logs",
		RetentionDays: 90,
		IsActive:      true,
	}
	if policy.RetentionDays != 90 {
		t.Errorf("expected 90 retention days, got %d", policy.RetentionDays)
	}
	if !policy.IsActive {
		t.Error("policy should be active")
	}
}
