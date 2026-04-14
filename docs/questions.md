# questions.md

## 1. Deployment Boundary and Network Isolation

**Question:** Must all features, integrations, and notifications operate strictly within the local network with zero outbound internet calls in all environments?
**Assumption:** Yes, all runtime paths are offline/LAN-only, including identity sync, token validation, ingestion, and webhook delivery.
**Solution:** Enforce a local-only network policy across backend and frontend configuration, disable external endpoints by default, and document allowed internal endpoint patterns.

---

## 2. Organization Hierarchy Model

**Question:** Is the organization hierarchy fixed to one predefined depth, or must administrators define custom labels and depth while preserving parent-child validation?
**Assumption:** Hierarchy depth and node labels are configurable per deployment, with strict parent-child integrity checks.
**Solution:** Implement a flexible tree schema with configurable level labels, uniqueness constraints per scope, and validation preventing cyclic or orphaned nodes.

---

## 3. Location Context Switching Rules

**Question:** When users switch location context, should all lists, KPIs, reports, and permission checks immediately scope to that context and inherited sub-locations?
**Assumption:** Context switch applies globally in-session and affects every data query and action authorization.
**Solution:** Add a centralized context resolver used by UI filters and backend authorization/multi-scope query guards.

---

## 4. Master Data Version Lifecycle Governance

**Question:** Which roles can draft, review, activate, and rollback master data versions, and is activation required to be single-effective per entity type?
**Assumption:** Draft and review are separated by role, and exactly one active effective version exists per entity type and scope.
**Solution:** Enforce workflow state transitions with role gates, transactional activation, and automatic deactivation of previous effective versions.

---

## 5. Concurrent Edit and Activation Conflict Policy

**Question:** How should concurrent edits or simultaneous activation attempts be resolved to prevent conflicting effective versions?
**Assumption:** Optimistic concurrency is required with explicit conflict errors and retry guidance.
**Solution:** Use version numbers/ETags and transactional checks to block stale updates and duplicate activation races.

---

## 6. Duplicate Detection Semantics

**Question:** What fields define duplicates for each master entity (SKU, color, size, season, brand, supplier, customer), and should matching be exact or normalized?
**Assumption:** Duplicate rules are entity-specific and use normalized comparisons (trim/case rules) where business-safe.
**Solution:** Define per-entity uniqueness rules, normalized indexing strategy, and clear duplicate warning payloads for UI display.

---

## 7. Deactivation Effect Boundaries

**Question:** Should deactivated records be hidden only from new selections while remaining visible in historical views, exports, and audits with status badges?
**Assumption:** Historical references remain intact and queryable; only new transactions are blocked from selecting deactivated items.
**Solution:** Implement soft-deactivation with reason, timestamp, actor, and selection guards that preserve referential history.

---

## 8. Bulk Import Contract

**Question:** Which file types, max file size, encoding, and template columns are required for local bulk import, and what is the row-level error reporting standard?
**Assumption:** CSV/XLSX imports are supported with deterministic column mapping and per-row validation feedback.
**Solution:** Provide import templates, strict parser rules, row-level error exports, and partial-success handling with auditable summaries.

---

## 9. Import Validation Rule Ownership

**Question:** Are validation rules globally fixed or configurable per source connector and entity with precedence rules?
**Assumption:** Source-level mapping and validation are configurable, with explicit precedence over defaults.
**Solution:** Implement rule sets with override precedence, schema validation, and versioned rule change history.

---

## 10. Connector Plugin Interface Definition

**Question:** What is the mandatory plugin contract (capabilities, lifecycle hooks, authentication mode, mapping interface, health checks) for folder/share/database connectors?
**Assumption:** Each connector must implement a common interface plus source-specific adapters.
**Solution:** Define a stable plugin interface with capability declarations, configuration schema validation, and connector health/status endpoints.

---

## 11. Incremental Load Cursor and Checkpoint Strategy

**Question:** Which cursor keys define incremental progress per source, and where/how are checkpoints stored to guarantee resume every 1,000 records?
**Assumption:** Checkpoints are durable, per job-run and per source, and survive service restarts.
**Solution:** Persist checkpoint metadata transactionally in MySQL, include last successful offset/token, and resume idempotently.

---

## 12. Backfill and Dependency-Orchestrated Scheduling

**Question:** How are full backfills prioritized against incremental jobs, and how should dependency failures propagate in the priority queue?
**Assumption:** Dependencies block downstream jobs; priority and fairness rules are deterministic and observable.
**Solution:** Implement queue policies for priority + dependency DAG execution with explicit blocked/ready/running/failed states.

---

## 13. Retry and Operator Acknowledgement Workflow

**Question:** After three retries with capped exponential backoff, what exact operator action is required to resume or rerun failed jobs?
**Assumption:** Manual acknowledgement with reason is mandatory before any further attempts.
**Solution:** Add a failure state requiring privileged acknowledgement, reason capture, and auditable rerun action.

---

## 14. Playback Media and Lyrics Input Rules

**Question:** Which audio formats and LRC variants are required, including encoding, timestamp precision, and optional word-level timing support?
**Assumption:** Common local formats are supported, and unknown lyric formats fail gracefully without breaking playback.
**Solution:** Implement format validation, robust LRC parser compatibility modes, and explicit fallback behavior for unsupported lyrics.

---

## 15. Lyrics Search-to-Seek Confirmation Definition

**Question:** What constitutes visible confirmation within 200 ms after search-based lyric seek (UI indicator, highlighted line, toast, or timeline jump marker)?
**Assumption:** Confirmation requires immediate UI state change tied to the nearest timestamp selection.
**Solution:** Add deterministic visual feedback on seek action and instrument client-side timing checks for the confirmation event.

---

## 16. Analytics KPI Source of Truth

**Question:** What are the canonical KPI formulas, aggregation windows, and refresh cadence, and which dimensions must be filterable by city/department scope?
**Assumption:** KPI definitions are centrally versioned and consistent across tiles, charts, and exports.
**Solution:** Implement a KPI definition registry with reusable query logic and scoped filter enforcement in every analytics endpoint.

---

## 17. Scheduled Report Execution Semantics

**Question:** Which timezone governs schedules like 06:00 AM, and what is the expected behavior for missed runs after downtime?
**Assumption:** Schedules use deployment-local timezone and support catch-up policy with explicit status tracking.
**Solution:** Add scheduler timezone configuration, missed-run policy (skip/catch-up), and job history states ready/failed with failure reasons.

---

## 18. CSV/PDF Export Fidelity and Access Control

**Question:** Must exported report content exactly match on-screen filters and user scope, and are downloads restricted by role and location permissions?
**Assumption:** Exports are permission-filtered snapshots matching selected filters at generation time.
**Solution:** Bind export generation to validated scope context and persist export metadata (requester, scope, filter hash, timestamp).

---

## 19. Authentication Mode and Session Policy Choice

**Question:** Should the implementation standardize on JWT, server sessions, or support both with a single source of truth for timeout enforcement?
**Assumption:** One primary mode is selected to avoid split security behavior; idle and max-session limits are enforced centrally.
**Solution:** Implement one authoritative auth strategy with middleware enforcing 30-minute idle and 12-hour hard expiry.

---

## 20. Password and Account Hardening

**Question:** What password complexity, lockout thresholds, and credential reset workflow are required for local username/password authentication?
**Assumption:** Minimum complexity and anti-bruteforce controls are mandatory in on-prem deployments.
**Solution:** Add password policy validation, progressive lockout, secure reset flow, and audit entries for login/security events.

---

## 21. RBAC and Data-Scope Authorization Matrix

**Question:** What is the explicit permission matrix by role for feature access plus city/department/location data scope at route, function, and object levels?
**Assumption:** Role checks alone are insufficient; object-level scope checks are required on every sensitive read/write.
**Solution:** Define a centralized authorization matrix and enforce it in API middleware and service-layer object ownership/scope guards.

---

## 22. Audit Log Immutability and Dual-Authorization Deletion

**Question:** How is audit-log immutability guaranteed, and what exact dual-authorization flow is required for deletion with recorded reason and timestamp?
**Assumption:** Audit records are append-only, with deletion as an exceptional controlled workflow requiring two distinct privileged approvers.
**Solution:** Use append-only storage policy, dual-approval workflow with independent identities, and immutable approval trail for deletion events.

---

## 23. Sensitive Data Classification and Masking Rules

**Question:** Which exact fields are classified as sensitive, what default masking format applies, and which roles can view unmasked values?
**Assumption:** Sensitive-field policy is explicit, role-aware, and consistently applied across UI, API responses, and exports.
**Solution:** Maintain a field-classification registry and enforce mask/unmask policies through shared response serializers.

---

## 24. Biometric Data Optionality and Key Management

**Question:** Is biometric storage optional per deployment, and if enabled, what key hierarchy, rotation procedure, and access controls are mandatory?
**Assumption:** Biometric features can be disabled entirely; if enabled, AES-256 encryption keys rotate every 90 days with least-privilege access.
**Solution:** Add feature flags for biometric workflows, local key management with rotation schedules, and strict key access auditing.

---

## 25. TLS Certificate and Internal Trust Policy

**Question:** What is the required TLS trust model for internal network traffic (self-signed/internal CA), and how are certificate renewal and pinning handled?
**Assumption:** Internal CA-based TLS is required for service-to-service and client-to-server traffic.
**Solution:** Define certificate issuance/renewal policy, enforce HTTPS-only endpoints, and document trust bootstrap for local clients.

---

## 26. Data Retention and Purge Safety

**Question:** Does the 30-day retention purge apply only to raw ingestion files or also to staged/intermediate artifacts, and what legal-hold exceptions are needed?
**Assumption:** Raw ingestion files are auto-purged by policy while preserving auditability metadata and legal-hold exclusions.
**Solution:** Implement retention scopes by artifact type, legal-hold overrides, purge logs, and dry-run previews for administrators.

---

## 27. Offline Federation and Directory Sync Boundaries

**Question:** For offline-compatible SSO/directory sync, what data elements, sync frequency, conflict resolution, and token validation source are required without external calls?
**Assumption:** Identity data is synchronized from local/on-prem identity sources with deterministic conflict resolution.
**Solution:** Define sync contracts, conflict policy, local token validation keys, and failure/recovery behavior for disconnected environments.

---

## 28. Internal Webhook/Message Delivery Guarantees

**Question:** What delivery semantics are required for internal notifications (at-most-once, at-least-once), and how are retries, deduplication, and signature verification handled?
**Assumption:** Reliable at-least-once delivery with idempotency and signed payloads is required for downstream on-prem integrations.
**Solution:** Implement queued event delivery with retry policy, idempotency keys, payload signing, and delivery audit status.

---

## 29. API Error Contract and Client Feedback Consistency

**Question:** What standard error schema and status-code mapping must all APIs follow for validation errors, auth failures, conflicts, and server faults?
**Assumption:** A unified error contract is required to keep frontend behavior consistent and auditable.
**Solution:** Define shared error response format with correlation IDs and enforce it via centralized middleware/handlers.

---

## 30. Static Acceptance Evidence Requirements

**Question:** Which concrete artifacts are required to prove completeness during static review (feature-to-module mapping, API docs, config samples, test mapping, security controls)?
**Assumption:** Static evidence must let reviewers trace each major requirement to implementation and tests without runtime execution.
**Solution:** Produce a concise evidence package in project docs linking requirements to routes/services/models/tests and operational controls.
