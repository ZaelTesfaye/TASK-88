-- =============================================================================
-- Multi-Org Data & Media Operations Hub - Database Schema
-- MySQL 8.0
--
-- NOTE: This file is for reference and fresh-bootstrap only.
-- In production, GORM AutoMigrate (backend/internal/database/database.go) is
-- the authoritative schema manager and runs at startup against the Go model
-- structs in backend/internal/models/. If the two diverge, the Go models win.
-- =============================================================================

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;
SET collation_connection = 'utf8mb4_unicode_ci';

-- -----------------------------------------------------------------------------
-- 1. users  (models.User)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username          VARCHAR(100)    NOT NULL,
    password_hash     VARCHAR(255)    NOT NULL,
    role              VARCHAR(50)     NOT NULL DEFAULT 'standard_user',
    city_scope        VARCHAR(255)    NOT NULL DEFAULT '',
    department_scope  VARCHAR(255)    NOT NULL DEFAULT '',
    status            VARCHAR(20)     NOT NULL DEFAULT 'active',
    failed_attempts   INT             NOT NULL DEFAULT 0,
    locked_until      DATETIME        NULL,
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_users_username (username),
    INDEX idx_users_role (role),
    INDEX idx_users_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Core user accounts – matches models.User';

-- -----------------------------------------------------------------------------
-- 2. sessions  (models.Session)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sessions (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id           BIGINT UNSIGNED NOT NULL,
    jwt_jti           VARCHAR(255)    NOT NULL,
    issued_at         DATETIME        NOT NULL,
    last_activity_at  DATETIME        NOT NULL,
    expires_at        DATETIME        NOT NULL,
    revoked_at        DATETIME        NULL,
    ip_address        VARCHAR(45)     NOT NULL DEFAULT '',
    user_agent        VARCHAR(512)    NOT NULL DEFAULT '',
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_sessions_jwt_jti (jwt_jti),
    INDEX idx_sessions_user_id (user_id),
    INDEX idx_sessions_expires_at (expires_at),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='JWT sessions – matches models.Session';

-- -----------------------------------------------------------------------------
-- 3. org_nodes  (models.OrgNode)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS org_nodes (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    parent_id       BIGINT UNSIGNED NULL,
    level_code      VARCHAR(50)     NOT NULL,
    level_label     VARCHAR(100)    NOT NULL,
    name            VARCHAR(255)    NOT NULL,
    city            VARCHAR(255)    NOT NULL DEFAULT '',
    department      VARCHAR(255)    NOT NULL DEFAULT '',
    is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
    sort_order      INT             NOT NULL DEFAULT 0,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_org_nodes_parent_id (parent_id),
    INDEX idx_org_nodes_level_code (level_code),
    INDEX idx_org_nodes_city (city),
    INDEX idx_org_nodes_department (department),
    INDEX idx_org_nodes_is_active (is_active),
    CONSTRAINT fk_org_nodes_parent FOREIGN KEY (parent_id) REFERENCES org_nodes(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Hierarchical org tree – matches models.OrgNode';

-- -----------------------------------------------------------------------------
-- 4. context_assignments  (models.ContextAssignment)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS context_assignments (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    org_node_id     BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_user_org (user_id, org_node_id),
    INDEX idx_context_assignments_org_node_id (org_node_id),
    CONSTRAINT fk_context_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_context_org FOREIGN KEY (org_node_id) REFERENCES org_nodes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Maps users to org nodes – matches models.ContextAssignment';

-- -----------------------------------------------------------------------------
-- 5. master_records  (models.MasterRecord)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS master_records (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    entity_type     VARCHAR(100)    NOT NULL,
    natural_key     VARCHAR(255)    NOT NULL,
    payload_json    JSON            NULL,
    status          VARCHAR(50)     NOT NULL DEFAULT 'active',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_entity_key (entity_type, natural_key),
    INDEX idx_master_records_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Master data records – matches models.MasterRecord';

-- -----------------------------------------------------------------------------
-- 6. master_versions  (models.MasterVersion)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS master_versions (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    entity_type     VARCHAR(100)    NOT NULL,
    scope_key       VARCHAR(255)    NOT NULL,
    version_no      INT             NOT NULL,
    state           VARCHAR(50)     NOT NULL DEFAULT 'draft',
    created_by      BIGINT UNSIGNED NOT NULL,
    reviewed_by     BIGINT UNSIGNED NULL,
    activated_by    BIGINT UNSIGNED NULL,
    activated_at    DATETIME        NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_version_entity_scope (entity_type, scope_key, version_no),
    INDEX idx_master_versions_state (state),
    CONSTRAINT fk_mversion_created_by FOREIGN KEY (created_by) REFERENCES users(id),
    CONSTRAINT fk_mversion_reviewed_by FOREIGN KEY (reviewed_by) REFERENCES users(id),
    CONSTRAINT fk_mversion_activated_by FOREIGN KEY (activated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Version snapshots – matches models.MasterVersion';

-- -----------------------------------------------------------------------------
-- 7. master_version_items  (models.MasterVersionItem)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS master_version_items (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    version_id        BIGINT UNSIGNED NOT NULL,
    master_record_id  BIGINT UNSIGNED NOT NULL,
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_version_record (version_id, master_record_id),
    CONSTRAINT fk_mvitem_version FOREIGN KEY (version_id) REFERENCES master_versions(id) ON DELETE CASCADE,
    CONSTRAINT fk_mvitem_record FOREIGN KEY (master_record_id) REFERENCES master_records(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Line items in a version – matches models.MasterVersionItem';

-- -----------------------------------------------------------------------------
-- 8. deactivation_events  (models.DeactivationEvent)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS deactivation_events (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    record_id       BIGINT UNSIGNED NOT NULL,
    reason          VARCHAR(1000)   NOT NULL,
    actor_user_id   BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_deactivation_events_record_id (record_id),
    INDEX idx_deactivation_events_actor_user_id (actor_user_id),
    CONSTRAINT fk_deact_record FOREIGN KEY (record_id) REFERENCES master_records(id) ON DELETE CASCADE,
    CONSTRAINT fk_deact_actor FOREIGN KEY (actor_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Deactivation history – matches models.DeactivationEvent';

-- -----------------------------------------------------------------------------
-- 9. import_sources  (models.ImportSource)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS import_sources (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name              VARCHAR(255)    NOT NULL,
    source_type       VARCHAR(100)    NOT NULL,
    connection_json   JSON            NULL,
    mapping_rules_json JSON           NULL,
    is_active         BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_import_sources_name (name),
    INDEX idx_import_sources_source_type (source_type),
    INDEX idx_import_sources_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Data import sources – matches models.ImportSource';

-- -----------------------------------------------------------------------------
-- 10. ingestion_jobs  (models.IngestionJob)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ingestion_jobs (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    import_source_id    BIGINT UNSIGNED NOT NULL,
    priority            INT             NOT NULL DEFAULT 0,
    state               VARCHAR(50)     NOT NULL DEFAULT 'pending',
    dependency_group    VARCHAR(255)    NOT NULL DEFAULT '',
    retry_count         INT             NOT NULL DEFAULT 0,
    max_retries         INT             NOT NULL DEFAULT 3,
    next_retry_at       DATETIME        NULL,
    total_records       INT             NOT NULL DEFAULT 0,
    processed_records   INT             NOT NULL DEFAULT 0,
    failed_records      INT             NOT NULL DEFAULT 0,
    acknowledged_by     BIGINT UNSIGNED NULL,
    acknowledged_reason VARCHAR(1000)   NOT NULL DEFAULT '',
    started_at          DATETIME        NULL,
    completed_at        DATETIME        NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_ingestion_jobs_import_source_id (import_source_id),
    INDEX idx_ingestion_jobs_priority (priority),
    INDEX idx_ingestion_jobs_state (state),
    INDEX idx_ingestion_jobs_dependency_group (dependency_group),
    INDEX idx_ingestion_jobs_next_retry_at (next_retry_at),
    CONSTRAINT fk_ingestion_source FOREIGN KEY (import_source_id) REFERENCES import_sources(id) ON DELETE CASCADE,
    CONSTRAINT fk_ingestion_ack FOREIGN KEY (acknowledged_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Import job runs – matches models.IngestionJob';

-- -----------------------------------------------------------------------------
-- 11. ingestion_checkpoints  (models.IngestionCheckpoint)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ingestion_checkpoints (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    job_id              BIGINT UNSIGNED NOT NULL,
    records_processed   INT             NOT NULL DEFAULT 0,
    cursor_token        VARCHAR(1000)   NOT NULL DEFAULT '',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_ingestion_checkpoints_job_id (job_id),
    CONSTRAINT fk_checkpoint_job FOREIGN KEY (job_id) REFERENCES ingestion_jobs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Checkpoint state for jobs – matches models.IngestionCheckpoint';

-- -----------------------------------------------------------------------------
-- 12. ingestion_failures  (models.IngestionFailure)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ingestion_failures (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    job_id          BIGINT UNSIGNED NOT NULL,
    record_index    INT             NOT NULL,
    raw_data        TEXT            NULL,
    error_message   VARCHAR(2000)   NOT NULL,
    error_code      VARCHAR(100)    NOT NULL DEFAULT '',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_ingestion_failures_job_id (job_id),
    INDEX idx_ingestion_failures_error_code (error_code),
    CONSTRAINT fk_failure_job FOREIGN KEY (job_id) REFERENCES ingestion_jobs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Row-level failure log – matches models.IngestionFailure';

-- -----------------------------------------------------------------------------
-- 13. media_assets  (models.MediaAsset)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS media_assets (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title           VARCHAR(500)    NOT NULL,
    audio_path      VARCHAR(1000)   NOT NULL DEFAULT '',
    cover_art_path  VARCHAR(1000)   NOT NULL DEFAULT '',
    theme_json      JSON            NULL,
    lyrics_lrc_path VARCHAR(1000)   NOT NULL DEFAULT '',
    duration        INT             NOT NULL DEFAULT 0,
    mime_type       VARCHAR(100)    NOT NULL DEFAULT '',
    file_size_bytes BIGINT          NOT NULL DEFAULT 0,
    uploaded_by     BIGINT UNSIGNED NULL,
    status          VARCHAR(50)     NOT NULL DEFAULT 'active',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_media_assets_title (title(191)),
    INDEX idx_media_assets_uploaded_by (uploaded_by),
    INDEX idx_media_assets_status (status),
    CONSTRAINT fk_media_user FOREIGN KEY (uploaded_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Media files – matches models.MediaAsset';

-- -----------------------------------------------------------------------------
-- 14. analytics_kpi_definitions  (models.AnalyticsKPIDefinition)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS analytics_kpi_definitions (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code            VARCHAR(100)    NOT NULL,
    display_name    VARCHAR(255)    NOT NULL,
    description     VARCHAR(1000)   NOT NULL DEFAULT '',
    formula_sql     TEXT            NOT NULL,
    dimensions_json JSON            NULL,
    is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_analytics_kpi_definitions_code (code),
    INDEX idx_analytics_kpi_definitions_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='KPI definitions – matches models.AnalyticsKPIDefinition';

-- -----------------------------------------------------------------------------
-- 15. report_schedules  (models.ReportSchedule)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS report_schedules (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name            VARCHAR(255)    NOT NULL,
    kpi_code        VARCHAR(100)    NOT NULL DEFAULT '',
    cron_expr       VARCHAR(100)    NOT NULL,
    timezone        VARCHAR(100)    NOT NULL DEFAULT 'America/New_York',
    output_format   VARCHAR(50)     NOT NULL DEFAULT 'xlsx',
    scope_json      JSON            NULL,
    recipients      VARCHAR(2000)   NOT NULL DEFAULT '',
    is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
    created_by      BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_report_schedules_kpi_code (kpi_code),
    INDEX idx_report_schedules_is_active (is_active),
    CONSTRAINT fk_report_sched_user FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Scheduled reports – matches models.ReportSchedule';

-- -----------------------------------------------------------------------------
-- 16. report_runs  (models.ReportRun)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS report_runs (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    schedule_id     BIGINT UNSIGNED NOT NULL,
    state           VARCHAR(50)     NOT NULL DEFAULT 'pending',
    output_path     VARCHAR(1000)   NOT NULL DEFAULT '',
    failure_reason  VARCHAR(2000)   NOT NULL DEFAULT '',
    requested_by    BIGINT UNSIGNED NULL,
    row_count       INT             NOT NULL DEFAULT 0,
    started_at      DATETIME        NULL,
    completed_at    DATETIME        NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_report_runs_schedule_id (schedule_id),
    INDEX idx_report_runs_state (state),
    CONSTRAINT fk_report_run_sched FOREIGN KEY (schedule_id) REFERENCES report_schedules(id) ON DELETE CASCADE,
    CONSTRAINT fk_report_run_user FOREIGN KEY (requested_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Report execution history – matches models.ReportRun';

-- -----------------------------------------------------------------------------
-- 17. audit_logs  (models.AuditLog)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    actor_user_id   BIGINT UNSIGNED NULL,
    action_type     VARCHAR(100)    NOT NULL,
    target_type     VARCHAR(100)    NOT NULL,
    target_id       VARCHAR(255)    NOT NULL,
    scope_json      JSON            NULL,
    sensitive_read  BOOLEAN         NOT NULL DEFAULT FALSE,
    metadata_json   JSON            NULL,
    ip_address      VARCHAR(45)     NOT NULL DEFAULT '',
    user_agent      VARCHAR(512)    NOT NULL DEFAULT '',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_audit_logs_actor_user_id (actor_user_id),
    INDEX idx_audit_logs_action_type (action_type),
    INDEX idx_audit_logs_target_type (target_type),
    INDEX idx_audit_logs_target_id (target_id),
    INDEX idx_audit_logs_sensitive_read (sensitive_read),
    INDEX idx_audit_logs_created_at (created_at),
    CONSTRAINT fk_audit_user FOREIGN KEY (actor_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Immutable audit trail – matches models.AuditLog';

-- -----------------------------------------------------------------------------
-- 18. audit_delete_requests  (models.AuditDeleteRequest)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_delete_requests (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    requested_by    BIGINT UNSIGNED NOT NULL,
    reason          VARCHAR(2000)   NOT NULL,
    state           VARCHAR(50)     NOT NULL DEFAULT 'pending',
    approver_one    BIGINT UNSIGNED NULL,
    approver_two    BIGINT UNSIGNED NULL,
    executed_at     DATETIME        NULL,
    target_type     VARCHAR(100)    NOT NULL,
    target_id       VARCHAR(255)    NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_audit_delete_requests_requested_by (requested_by),
    INDEX idx_audit_delete_requests_state (state),
    CONSTRAINT fk_audit_del_requested FOREIGN KEY (requested_by) REFERENCES users(id),
    CONSTRAINT fk_audit_del_approver1 FOREIGN KEY (approver_one) REFERENCES users(id),
    CONSTRAINT fk_audit_del_approver2 FOREIGN KEY (approver_two) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Dual-approval audit deletion – matches models.AuditDeleteRequest';

-- -----------------------------------------------------------------------------
-- 19. sensitive_field_registry  (models.SensitiveFieldRegistry)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sensitive_field_registry (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    field_key         VARCHAR(255)    NOT NULL,
    display_name      VARCHAR(255)    NOT NULL DEFAULT '',
    mask_pattern      VARCHAR(255)    NOT NULL,
    unmask_roles_json JSON            NULL,
    is_active         BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_sensitive_field_registry_field_key (field_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='PII field registry – matches models.SensitiveFieldRegistry';

-- -----------------------------------------------------------------------------
-- 20. key_rings  (models.KeyRing – note: table name is key_rings)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS key_rings (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    key_id          VARCHAR(255)    NOT NULL,
    key_purpose     VARCHAR(100)    NOT NULL,
    wrapped_key     VARCHAR(4000)   NOT NULL,
    algorithm       VARCHAR(50)     NOT NULL DEFAULT 'AES-256-GCM',
    rotates_at      DATETIME        NOT NULL,
    status          VARCHAR(50)     NOT NULL DEFAULT 'active',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_key_rings_key_id (key_id),
    INDEX idx_key_rings_key_purpose (key_purpose),
    INDEX idx_key_rings_rotates_at (rotates_at),
    INDEX idx_key_rings_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Encryption key ring – matches models.KeyRing';

-- -----------------------------------------------------------------------------
-- 21. password_reset_requests  (models.PasswordResetRequest)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS password_reset_requests (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    requested_by    BIGINT UNSIGNED NOT NULL,
    approved_by     BIGINT UNSIGNED NULL,
    token_hash      VARCHAR(255)    NOT NULL DEFAULT '',
    expires_at      DATETIME        NOT NULL,
    used_at         DATETIME        NULL,
    status          VARCHAR(50)     NOT NULL DEFAULT 'pending',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_password_reset_requests_user_id (user_id),
    INDEX idx_password_reset_requests_expires_at (expires_at),
    INDEX idx_password_reset_requests_status (status),
    CONSTRAINT fk_pwreset_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_pwreset_requested FOREIGN KEY (requested_by) REFERENCES users(id),
    CONSTRAINT fk_pwreset_approved FOREIGN KEY (approved_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Password reset tokens – matches models.PasswordResetRequest';

-- -----------------------------------------------------------------------------
-- 22. retention_policies  (models.RetentionPolicy)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS retention_policies (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    artifact_type       VARCHAR(100)    NOT NULL,
    retention_days      INT             NOT NULL,
    legal_hold_enabled  BOOLEAN         NOT NULL DEFAULT FALSE,
    description         VARCHAR(1000)   NOT NULL DEFAULT '',
    is_active           BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_retention_policies_artifact_type (artifact_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Data retention rules – matches models.RetentionPolicy';

-- -----------------------------------------------------------------------------
-- 23. legal_holds  (models.LegalHold)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS legal_holds (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    scope_json      JSON            NOT NULL,
    reason          VARCHAR(2000)   NOT NULL,
    created_by      BIGINT UNSIGNED NOT NULL,
    released_at     DATETIME        NULL,
    released_by     BIGINT UNSIGNED NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_legal_holds_created_by (created_by),
    CONSTRAINT fk_legal_hold_created FOREIGN KEY (created_by) REFERENCES users(id),
    CONSTRAINT fk_legal_hold_released FOREIGN KEY (released_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Legal hold records – matches models.LegalHold';

-- -----------------------------------------------------------------------------
-- 24. purge_runs  (models.PurgeRun)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS purge_runs (
    id                          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    artifact_type               VARCHAR(100)    NOT NULL,
    dry_run                     BOOLEAN         NOT NULL DEFAULT TRUE,
    purged_count                INT             NOT NULL DEFAULT 0,
    blocked_by_legal_hold_count INT             NOT NULL DEFAULT 0,
    started_at                  DATETIME        NULL,
    completed_at                DATETIME        NULL,
    initiated_by                BIGINT UNSIGNED NULL,
    created_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_purge_runs_artifact_type (artifact_type),
    CONSTRAINT fk_purge_initiator FOREIGN KEY (initiated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Purge execution log – matches models.PurgeRun';

-- -----------------------------------------------------------------------------
-- 25. integration_endpoints  (models.IntegrationEndpoint)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS integration_endpoints (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name              VARCHAR(255)    NOT NULL,
    event_type        VARCHAR(100)    NOT NULL,
    url               VARCHAR(2000)   NOT NULL,
    signing_secret_ref VARCHAR(255)   NOT NULL DEFAULT '',
    http_method       VARCHAR(10)     NOT NULL DEFAULT 'POST',
    headers_json      JSON            NULL,
    enabled           BOOLEAN         NOT NULL DEFAULT TRUE,
    max_retries       INT             NOT NULL DEFAULT 3,
    timeout_seconds   INT             NOT NULL DEFAULT 30,
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_integration_endpoints_event_type (event_type),
    INDEX idx_integration_endpoints_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Webhook/API endpoints – matches models.IntegrationEndpoint';

-- -----------------------------------------------------------------------------
-- 26. integration_deliveries  (models.IntegrationDelivery)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS integration_deliveries (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    endpoint_id     BIGINT UNSIGNED NOT NULL,
    event_id        VARCHAR(255)    NOT NULL,
    state           VARCHAR(50)     NOT NULL DEFAULT 'pending',
    payload_json    TEXT            NULL,
    response_code   INT             NULL,
    response_body   TEXT            NULL,
    retries         INT             NOT NULL DEFAULT 0,
    next_retry_at   DATETIME        NULL,
    dedupe_key      VARCHAR(255)    NOT NULL DEFAULT '',
    delivered_at    DATETIME        NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_integration_deliveries_endpoint_id (endpoint_id),
    INDEX idx_integration_deliveries_event_id (event_id),
    INDEX idx_integration_deliveries_state (state),
    INDEX idx_integration_deliveries_next_retry_at (next_retry_at),
    INDEX idx_integration_deliveries_dedupe_key (dedupe_key),
    CONSTRAINT fk_delivery_endpoint FOREIGN KEY (endpoint_id) REFERENCES integration_endpoints(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Outbound delivery log – matches models.IntegrationDelivery';

-- -----------------------------------------------------------------------------
-- 27. connector_definitions  (models.ConnectorDefinition)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS connector_definitions (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name                VARCHAR(255)    NOT NULL,
    connector_type      VARCHAR(100)    NOT NULL,
    capabilities_json   JSON            NULL,
    config_schema_json  JSON            NULL,
    health_status       VARCHAR(50)     NOT NULL DEFAULT 'unknown',
    last_health_check_at DATETIME       NULL,
    is_active           BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_connector_definitions_connector_type (connector_type),
    INDEX idx_connector_definitions_health_status (health_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Connector templates – matches models.ConnectorDefinition';


-- =============================================================================
-- SEED DATA
-- =============================================================================

-- Default admin user (password: Admin@12345678)
-- Argon2id hash – the application uses Argon2id (see backend/internal/auth/auth_service.go)
INSERT INTO users (username, password_hash, role, city_scope, department_scope, status)
VALUES (
    'admin',
    '$argon2id$v=19$m=65536,t=1,p=4$c29tZXNhbHQ$hash_placeholder',
    'system_admin',
    '*',
    '*',
    'active'
);

-- Default root organisation node
INSERT INTO org_nodes (level_code, level_label, name)
VALUES ('company', 'Company', 'Root Organisation');

-- Assign admin to root org
INSERT INTO context_assignments (user_id, org_node_id)
VALUES (1, 1);

-- Default sensitive field registry entries
INSERT INTO sensitive_field_registry (field_key, display_name, mask_pattern, unmask_roles_json) VALUES
('users.email', 'User Email', 'email', '["system_admin"]'),
('users.password_hash', 'Password Hash', 'full', '[]'),
('sessions.ip_address', 'Session IP', 'last4', '["system_admin"]'),
('key_rings.wrapped_key', 'Encryption Key', 'full', '[]');

-- Default retention policies
INSERT INTO retention_policies (artifact_type, retention_days, description) VALUES
('audit_logs', 730, 'Audit trail retention'),
('sessions', 30, 'Expired session cleanup'),
('ingestion_failures', 90, 'Failed ingestion record cleanup'),
('password_reset_requests', 7, 'Expired reset token cleanup'),
('report_runs', 365, 'Old report run cleanup'),
('integration_deliveries', 180, 'Delivery log cleanup');

-- Default KPI definitions
INSERT INTO analytics_kpi_definitions (code, display_name, description, formula_sql, dimensions_json) VALUES
('master_record_count', 'Master Records', 'Total master records created in period', 'SELECT COUNT(*) FROM master_records WHERE created_at BETWEEN ? AND ?', '{"unit":"count"}'),
('active_record_count', 'Active Records', 'Currently active master records', 'SELECT COUNT(*) FROM master_records WHERE status = ''active''', '{"unit":"count"}'),
('ingestion_job_count', 'Ingestion Jobs', 'Ingestion jobs run in period', 'SELECT COUNT(*) FROM ingestion_jobs WHERE created_at BETWEEN ? AND ?', '{"unit":"count"}'),
('ingestion_success_rate', 'Ingestion Success Rate', 'Percentage of successful ingestion jobs', '', '{"unit":"percent","rate":true}'),
('ingestion_failure_count', 'Ingestion Failures', 'Row-level failures in period', 'SELECT COUNT(*) FROM ingestion_failures WHERE created_at BETWEEN ? AND ?', '{"unit":"count"}'),
('report_run_count', 'Report Runs', 'Report runs in period', 'SELECT COUNT(*) FROM report_runs WHERE created_at BETWEEN ? AND ?', '{"unit":"count"}'),
('report_completion_rate', 'Report Completion Rate', 'Percentage of reports completed', '', '{"unit":"percent","rate":true}'),
('version_count', 'Versions Created', 'Master data versions created in period', 'SELECT COUNT(*) FROM master_versions WHERE created_at BETWEEN ? AND ?', '{"unit":"count"}'),
('active_user_count', 'Active Users', 'Currently active user accounts', 'SELECT COUNT(*) FROM users WHERE status = ''active''', '{"unit":"count"}'),
('audit_event_count', 'Audit Events', 'Audit log entries in period', 'SELECT COUNT(*) FROM audit_logs WHERE created_at BETWEEN ? AND ?', '{"unit":"count"}');
