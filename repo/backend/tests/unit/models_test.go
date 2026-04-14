package unit

import (
	"testing"

	"backend/internal/models"
)

// Every model's TableName() must return the expected table name.
func TestAllModelTableNames(t *testing.T) {
	cases := []struct {
		name     string
		got      string
		expected string
	}{
		{"User", models.User{}.TableName(), "users"},
		{"Session", models.Session{}.TableName(), "sessions"},
		{"OrgNode", models.OrgNode{}.TableName(), "org_nodes"},
		{"ContextAssignment", models.ContextAssignment{}.TableName(), "context_assignments"},
		{"MasterRecord", models.MasterRecord{}.TableName(), "master_records"},
		{"MasterVersion", models.MasterVersion{}.TableName(), "master_versions"},
		{"MasterVersionItem", models.MasterVersionItem{}.TableName(), "master_version_items"},
		{"DeactivationEvent", models.DeactivationEvent{}.TableName(), "deactivation_events"},
		{"ImportSource", models.ImportSource{}.TableName(), "import_sources"},
		{"IngestionJob", models.IngestionJob{}.TableName(), "ingestion_jobs"},
		{"IngestionCheckpoint", models.IngestionCheckpoint{}.TableName(), "ingestion_checkpoints"},
		{"IngestionFailure", models.IngestionFailure{}.TableName(), "ingestion_failures"},
		{"MediaAsset", models.MediaAsset{}.TableName(), "media_assets"},
		{"AnalyticsKPIDefinition", models.AnalyticsKPIDefinition{}.TableName(), "analytics_kpi_definitions"},
		{"ReportSchedule", models.ReportSchedule{}.TableName(), "report_schedules"},
		{"ReportRun", models.ReportRun{}.TableName(), "report_runs"},
		{"AuditLog", models.AuditLog{}.TableName(), "audit_logs"},
		{"AuditDeleteRequest", models.AuditDeleteRequest{}.TableName(), "audit_delete_requests"},
		{"SensitiveFieldRegistry", models.SensitiveFieldRegistry{}.TableName(), "sensitive_field_registry"},
		{"KeyRing", models.KeyRing{}.TableName(), "key_rings"},
		{"PasswordResetRequest", models.PasswordResetRequest{}.TableName(), "password_reset_requests"},
		{"RetentionPolicy", models.RetentionPolicy{}.TableName(), "retention_policies"},
		{"LegalHold", models.LegalHold{}.TableName(), "legal_holds"},
		{"PurgeRun", models.PurgeRun{}.TableName(), "purge_runs"},
		{"IntegrationEndpoint", models.IntegrationEndpoint{}.TableName(), "integration_endpoints"},
		{"IntegrationDelivery", models.IntegrationDelivery{}.TableName(), "integration_deliveries"},
		{"ConnectorDefinition", models.ConnectorDefinition{}.TableName(), "connector_definitions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("TableName() = %q, want %q", tc.got, tc.expected)
			}
		})
	}
}
