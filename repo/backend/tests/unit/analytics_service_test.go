package unit

import (
	"testing"

	"backend/internal/analytics"
)

// Tests for the real analytics/analytics_service.go production code.

func TestNewAnalyticsServiceCreation(t *testing.T) {
	svc := analytics.NewAnalyticsService(nil)
	if svc == nil {
		t.Fatal("expected non-nil AnalyticsService")
	}
}

func TestKPIFilterStructDefaults(t *testing.T) {
	f := analytics.KPIFilter{}
	if f.CityScope != "" {
		t.Errorf("expected empty CityScope, got %q", f.CityScope)
	}
	if f.DeptScope != "" {
		t.Errorf("expected empty DeptScope, got %q", f.DeptScope)
	}
	if len(f.NodeIDs) != 0 {
		t.Errorf("expected empty NodeIDs, got %v", f.NodeIDs)
	}
	if f.DateFrom != "" {
		t.Errorf("expected empty DateFrom, got %q", f.DateFrom)
	}
	if f.DateTo != "" {
		t.Errorf("expected empty DateTo, got %q", f.DateTo)
	}
}

func TestTrendFilterGranularityValues(t *testing.T) {
	for _, g := range []string{"daily", "weekly", "monthly"} {
		f := analytics.TrendFilter{Granularity: g}
		if f.Granularity != g {
			t.Errorf("expected %q, got %q", g, f.Granularity)
		}
	}
}

func TestKPIResultStructFields(t *testing.T) {
	r := analytics.KPIResult{
		Code: "sku_velocity", Label: "SKU Velocity",
		Value: 42.5, PrevValue: 40.0, ChangePercent: 6.25,
		TrendDirection: "up", Unit: "units/day",
	}
	if r.Code != "sku_velocity" {
		t.Errorf("unexpected Code: %v", r.Code)
	}
	if r.TrendDirection != "up" {
		t.Errorf("expected trend up, got %v", r.TrendDirection)
	}
	if r.ChangePercent != 6.25 {
		t.Errorf("expected ChangePercent 6.25, got %v", r.ChangePercent)
	}
}

func TestTrendSeriesStructure(t *testing.T) {
	s := analytics.TrendSeries{
		Code:  "fill_rate",
		Label: "Fill Rate",
		Points: []analytics.DataPoint{
			{Date: "2025-01-01", Value: 92.5},
			{Date: "2025-01-02", Value: 93.1},
		},
	}
	if len(s.Points) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(s.Points))
	}
	if s.Points[0].Date != "2025-01-01" {
		t.Errorf("expected first point date 2025-01-01, got %v", s.Points[0].Date)
	}
}

func TestGetKPIDefinitionsWithNilDBPanics(t *testing.T) {
	svc := analytics.NewAnalyticsService(nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when querying with nil DB")
		}
	}()
	svc.GetKPIDefinitions()
}

func TestGetKPIDefinitionByCodeWithNilDBPanics(t *testing.T) {
	svc := analytics.NewAnalyticsService(nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when querying with nil DB")
		}
	}()
	svc.GetKPIDefinitionByCode("test")
}
