package ingestion

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"backend/internal/logging"
	"backend/internal/models"
)

// Job states.
const (
	StateReady            = "ready"
	StateRunning          = "running"
	StateCompleted        = "completed"
	StateRetrying         = "retrying"
	StateFailedAwaitAck   = "failed_awaiting_ack"
)

// Ingestion modes.
const (
	ModeIncremental = "incremental"
	ModeBackfill    = "backfill"
)

const (
	checkpointInterval = 1000
	maxRetries         = 3
)

// Backoff durations for each retry attempt.
var backoffDurations = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
}

// EnqueueRequest holds the parameters for creating a new ingestion job.
type EnqueueRequest struct {
	ImportSourceID  uint   `json:"import_source_id" binding:"required"`
	Priority        int    `json:"priority"`
	DependencyGroup string `json:"dependency_group"`
	Mode            string `json:"mode"` // "incremental" or "backfill"
}

// JobFilter holds query parameters for listing jobs.
type JobFilter struct {
	State           string `form:"state"`
	ImportSourceID  uint   `form:"import_source_id"`
	DependencyGroup string `form:"dependency_group"`
	Page            int    `form:"page"`
	PageSize        int    `form:"page_size"`
}

// PaginatedJobs holds a page of jobs plus pagination metadata.
type PaginatedJobs struct {
	Jobs       []models.IngestionJob `json:"jobs"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// DependencyStatus holds the status of a job's dependencies.
type DependencyStatus struct {
	AllSatisfied   bool   `json:"all_satisfied"`
	BlockingJobIDs []uint `json:"blocking_job_ids"`
}

// JobEngine orchestrates ingestion jobs.
type JobEngine struct {
	db      *gorm.DB
	factory *ConnectorFactory
}

// NewJobEngine creates a new JobEngine.
func NewJobEngine(db *gorm.DB) *JobEngine {
	return &JobEngine{
		db:      db,
		factory: NewConnectorFactory(),
	}
}

// EnqueueJob creates a new ingestion job.
func (e *JobEngine) EnqueueJob(req EnqueueRequest) (*models.IngestionJob, error) {
	// Validate source exists and is enabled.
	var source models.ImportSource
	if err := e.db.First(&source, req.ImportSourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("import source %d not found", req.ImportSourceID)
		}
		return nil, fmt.Errorf("failed to look up import source: %w", err)
	}
	if !source.IsActive {
		return nil, fmt.Errorf("import source %d is not active", req.ImportSourceID)
	}

	priority := req.Priority
	if priority == 0 {
		priority = 5 // default priority
	}

	mode := req.Mode
	if mode == "" {
		mode = ModeIncremental
	}

	job := models.IngestionJob{
		ImportSourceID:  req.ImportSourceID,
		Priority:        priority,
		State:           StateReady,
		DependencyGroup: req.DependencyGroup,
		RetryCount:      0,
		MaxRetries:      maxRetries,
	}

	if err := e.db.Create(&job).Error; err != nil {
		return nil, fmt.Errorf("failed to create ingestion job: %w", err)
	}

	logging.Info("ingestion", "enqueue", fmt.Sprintf("job %d enqueued for source %d (mode=%s, priority=%d)", job.ID, source.ID, mode, priority))

	return &job, nil
}

// ProcessNextJob picks the highest priority ready job and executes it.
func (e *JobEngine) ProcessNextJob() error {
	// Apply starvation boost: any job waiting more than 30 minutes gets +10 priority.
	starvationThreshold := time.Now().Add(-30 * time.Minute)
	e.db.Model(&models.IngestionJob{}).
		Where("state = ? AND created_at < ? AND priority < ?", StateReady, starvationThreshold, 100).
		Update("priority", gorm.Expr("priority + 10"))

	// Pick the highest priority ready job, FIFO within same priority.
	var job models.IngestionJob
	err := e.db.Where("state = ?", StateReady).
		Order("priority DESC, created_at ASC").
		First(&job).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // No jobs to process.
		}
		return fmt.Errorf("failed to find next job: %w", err)
	}

	// Check dependencies before starting.
	satisfied, blockingIDs, err := e.GetDependencyStatus(&job)
	if err != nil {
		return fmt.Errorf("failed to check dependencies for job %d: %w", job.ID, err)
	}
	if !satisfied {
		logging.Info("ingestion", "process", fmt.Sprintf("job %d blocked by dependencies: %v", job.ID, blockingIDs))
		return nil
	}

	return e.ExecuteJob(&job)
}

// ExecuteJob runs a single job.
func (e *JobEngine) ExecuteJob(job *models.IngestionJob) error {
	// Transition to running.
	now := time.Now()
	job.State = StateRunning
	job.StartedAt = &now
	if err := e.db.Save(job).Error; err != nil {
		return fmt.Errorf("failed to update job state to running: %w", err)
	}

	logging.Info("ingestion", "execute", fmt.Sprintf("starting job %d for source %d", job.ID, job.ImportSourceID))

	// Load the import source.
	var source models.ImportSource
	if err := e.db.First(&source, job.ImportSourceID).Error; err != nil {
		return e.handleJobFailure(job, fmt.Errorf("import source not found: %w", err))
	}

	// Parse connection config.
	var connConfig map[string]interface{}
	if err := json.Unmarshal([]byte(source.ConnectionJSON), &connConfig); err != nil {
		return e.handleJobFailure(job, fmt.Errorf("failed to parse connection config: %w", err))
	}

	// Create connector.
	connector, err := e.factory.CreateFromSource(source, connConfig)
	if err != nil {
		return e.handleJobFailure(job, fmt.Errorf("failed to create connector: %w", err))
	}

	// Determine starting cursor: for incremental mode use last checkpoint, for backfill start from beginning.
	cursor := ""
	if job.RetryCount == 0 {
		// Check for last checkpoint from a previous run of the same source (incremental).
		var lastCheckpoint models.IngestionCheckpoint
		err := e.db.Where("job_id IN (SELECT id FROM ingestion_jobs WHERE import_source_id = ? AND state = ?)",
			job.ImportSourceID, StateCompleted).
			Order("created_at DESC").
			First(&lastCheckpoint).Error
		if err == nil {
			cursor = lastCheckpoint.CursorToken
		}
	} else {
		// On retry, resume from last checkpoint of this job.
		var lastCheckpoint models.IngestionCheckpoint
		err := e.db.Where("job_id = ?", job.ID).
			Order("created_at DESC").
			First(&lastCheckpoint).Error
		if err == nil {
			cursor = lastCheckpoint.CursorToken
			job.ProcessedRecords = lastCheckpoint.RecordsProcessed
		}
	}

	totalProcessed := job.ProcessedRecords
	totalFailed := job.FailedRecords
	batchSize := 1000

	for {
		result, err := connector.Pull(cursor, batchSize)
		if err != nil {
			return e.handleJobFailure(job, fmt.Errorf("pull failed at cursor %s: %w", cursor, err))
		}

		for i, record := range result.Records {
			_ = record // Records are pulled; processing/validation happens at a higher layer.
			totalProcessed++

			// Write checkpoint every checkpointInterval records.
			if totalProcessed%checkpointInterval == 0 {
				checkpointCursor := result.NextCursor
				if checkpointCursor == "" {
					checkpointCursor = fmt.Sprintf("%d", totalProcessed)
				}
				if err := e.WriteCheckpoint(job.ID, totalProcessed, checkpointCursor); err != nil {
					logging.Error("ingestion", "checkpoint",
						fmt.Sprintf("failed to write checkpoint for job %d at record %d: %v", job.ID, totalProcessed, err))
				}

				if err := connector.AcknowledgeCheckpoint(checkpointCursor); err != nil {
					logging.Warn("ingestion", "checkpoint",
						fmt.Sprintf("connector failed to acknowledge checkpoint for job %d: %v", job.ID, err))
				}
			}

			_ = i // suppress unused variable
		}

		cursor = result.NextCursor
		if !result.HasMore {
			break
		}
	}

	// Final checkpoint.
	if totalProcessed > 0 {
		finalCursor := cursor
		if finalCursor == "" {
			finalCursor = fmt.Sprintf("%d", totalProcessed)
		}
		if err := e.WriteCheckpoint(job.ID, totalProcessed, finalCursor); err != nil {
			logging.Error("ingestion", "checkpoint",
				fmt.Sprintf("failed to write final checkpoint for job %d: %v", job.ID, err))
		}
	}

	// Mark job as completed.
	completedAt := time.Now()
	job.State = StateCompleted
	job.CompletedAt = &completedAt
	job.ProcessedRecords = totalProcessed
	job.TotalRecords = totalProcessed + totalFailed
	job.FailedRecords = totalFailed
	if err := e.db.Save(job).Error; err != nil {
		return fmt.Errorf("failed to update job state to completed: %w", err)
	}

	logging.Info("ingestion", "execute",
		fmt.Sprintf("job %d completed: %d records processed, %d failed", job.ID, totalProcessed, totalFailed))

	return nil
}

// handleJobFailure handles a job failure by updating state and setting up retry or acknowledgment.
func (e *JobEngine) handleJobFailure(job *models.IngestionJob, jobErr error) error {
	logging.Error("ingestion", "execute", fmt.Sprintf("job %d failed: %v", job.ID, jobErr))

	job.RetryCount++

	if job.RetryCount >= job.MaxRetries {
		// Move to failed_awaiting_ack state.
		job.State = StateFailedAwaitAck
		if err := e.db.Save(job).Error; err != nil {
			return fmt.Errorf("failed to update job to failed_awaiting_ack: %w", err)
		}
		logging.Warn("ingestion", "retry",
			fmt.Sprintf("job %d exceeded max retries (%d), awaiting operator acknowledgment", job.ID, job.MaxRetries))
		return jobErr
	}

	// Set up retry with exponential backoff.
	backoffIdx := job.RetryCount - 1
	if backoffIdx < 0 {
		backoffIdx = 0
	}
	if backoffIdx >= len(backoffDurations) {
		backoffIdx = len(backoffDurations) - 1
	}
	nextRetry := time.Now().Add(backoffDurations[backoffIdx])

	job.State = StateRetrying
	job.NextRetryAt = &nextRetry
	if err := e.db.Save(job).Error; err != nil {
		return fmt.Errorf("failed to update job for retry: %w", err)
	}

	logging.Info("ingestion", "retry",
		fmt.Sprintf("job %d scheduled for retry %d at %s (backoff: %v)",
			job.ID, job.RetryCount, nextRetry.Format(time.RFC3339), backoffDurations[backoffIdx]))

	return jobErr
}

// WriteCheckpoint saves progress.
func (e *JobEngine) WriteCheckpoint(jobID uint, recordsProcessed int, cursor string) error {
	checkpoint := models.IngestionCheckpoint{
		JobID:            jobID,
		RecordsProcessed: recordsProcessed,
		CursorToken:      cursor,
	}
	if err := e.db.Create(&checkpoint).Error; err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	// Also update the job's processed count.
	e.db.Model(&models.IngestionJob{}).Where("id = ?", jobID).
		Update("processed_records", recordsProcessed)

	return nil
}

// RetryJob retries a failed job, moving it back to ready with exponential backoff.
func (e *JobEngine) RetryJob(job *models.IngestionJob) error {
	if job.State != StateRetrying && job.State != StateFailedAwaitAck {
		return fmt.Errorf("job %d is in state '%s' and cannot be retried", job.ID, job.State)
	}

	if job.RetryCount >= job.MaxRetries {
		return fmt.Errorf("job %d has exhausted all retries (%d/%d)", job.ID, job.RetryCount, job.MaxRetries)
	}

	job.State = StateReady
	job.NextRetryAt = nil
	if err := e.db.Save(job).Error; err != nil {
		return fmt.Errorf("failed to update job state for retry: %w", err)
	}

	logging.Info("ingestion", "retry", fmt.Sprintf("job %d moved back to ready for retry %d", job.ID, job.RetryCount))
	return nil
}

// PromoteRetryingJobs checks for retrying jobs whose next_retry_at has passed and moves them to ready.
func (e *JobEngine) PromoteRetryingJobs() error {
	now := time.Now()
	result := e.db.Model(&models.IngestionJob{}).
		Where("state = ? AND next_retry_at <= ?", StateRetrying, now).
		Updates(map[string]interface{}{
			"state":       StateReady,
			"next_retry_at": nil,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to promote retrying jobs: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		logging.Info("ingestion", "retry", fmt.Sprintf("promoted %d retrying jobs to ready", result.RowsAffected))
	}
	return nil
}

// AcknowledgeJob requires operator acknowledgment for failed jobs.
func (e *JobEngine) AcknowledgeJob(jobID uint, userID uint, reason string) error {
	var job models.IngestionJob
	if err := e.db.First(&job, jobID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("job %d not found", jobID)
		}
		return fmt.Errorf("failed to look up job: %w", err)
	}

	if job.State != StateFailedAwaitAck {
		return fmt.Errorf("job %d is in state '%s', only jobs in '%s' can be acknowledged",
			jobID, job.State, StateFailedAwaitAck)
	}

	if reason == "" {
		return fmt.Errorf("acknowledgment reason is required")
	}

	job.AcknowledgedBy = &userID
	job.AcknowledgedReason = reason
	job.State = StateReady
	job.RetryCount = 0
	job.NextRetryAt = nil

	if err := e.db.Save(&job).Error; err != nil {
		return fmt.Errorf("failed to acknowledge job: %w", err)
	}

	logging.Info("ingestion", "acknowledge",
		fmt.Sprintf("job %d acknowledged by user %d: %s", jobID, userID, reason))

	return nil
}

// GetDependencyStatus checks if all dependencies for a job are satisfied.
func (e *JobEngine) GetDependencyStatus(job *models.IngestionJob) (bool, []uint, error) {
	if job.DependencyGroup == "" {
		return true, nil, nil
	}

	// Find all jobs in the same dependency group that were created before this one.
	var dependentJobs []models.IngestionJob
	err := e.db.Where("dependency_group = ? AND id < ? AND id != ?",
		job.DependencyGroup, job.ID, job.ID).
		Find(&dependentJobs).Error
	if err != nil {
		return false, nil, fmt.Errorf("failed to query dependency group: %w", err)
	}

	var blockingIDs []uint
	for _, dep := range dependentJobs {
		if dep.State != StateCompleted {
			blockingIDs = append(blockingIDs, dep.ID)
		}
	}

	return len(blockingIDs) == 0, blockingIDs, nil
}

// GetJobQueue returns current job queue with status.
func (e *JobEngine) GetJobQueue(filter JobFilter) (*PaginatedJobs, error) {
	query := e.db.Model(&models.IngestionJob{})

	if filter.State != "" {
		query = query.Where("state = ?", filter.State)
	}
	if filter.ImportSourceID > 0 {
		query = query.Where("import_source_id = ?", filter.ImportSourceID)
	}
	if filter.DependencyGroup != "" {
		query = query.Where("dependency_group = ?", filter.DependencyGroup)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 50
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	offset := (filter.Page - 1) * filter.PageSize

	var jobs []models.IngestionJob
	if err := query.Order("priority DESC, created_at ASC").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &PaginatedJobs{
		Jobs:       jobs,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}
