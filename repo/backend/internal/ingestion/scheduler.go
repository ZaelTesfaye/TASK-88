package ingestion

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"backend/internal/logging"
	"backend/internal/models"
)

// scheduledSource tracks cron entries for import sources.
type scheduledSource struct {
	sourceID uint
	entryID  cron.EntryID
	cronExpr string
}

// Scheduler manages periodic ingestion job scheduling.
type Scheduler struct {
	db       *gorm.DB
	engine   *JobEngine
	cron     *cron.Cron
	mu       sync.Mutex
	sources  map[uint]*scheduledSource
	stopChan chan struct{}
}

// NewScheduler creates a new Scheduler.
func NewScheduler(db *gorm.DB, engine *JobEngine) *Scheduler {
	return &Scheduler{
		db:      db,
		engine:  engine,
		cron:    cron.New(cron.WithSeconds()),
		sources: make(map[uint]*scheduledSource),
	}
}

// Start begins the scheduler. It loads active sources, schedules them,
// handles missed runs, and starts the internal cron runner plus a retry promoter.
func (s *Scheduler) Start() error {
	logging.Info("ingestion", "scheduler", "starting ingestion scheduler")

	// Handle any missed runs from downtime.
	if err := s.HandleMissedRuns(); err != nil {
		logging.Error("ingestion", "scheduler", fmt.Sprintf("failed to handle missed runs: %v", err))
	}

	// Start the retry promoter: checks every 30 seconds for retrying jobs ready to re-run.
	s.cron.AddFunc("*/30 * * * * *", func() {
		if err := s.engine.PromoteRetryingJobs(); err != nil {
			logging.Error("ingestion", "scheduler", fmt.Sprintf("retry promotion failed: %v", err))
		}
	})

	// Start the job processor: runs every 10 seconds.
	s.cron.AddFunc("*/10 * * * * *", func() {
		if err := s.engine.ProcessNextJob(); err != nil {
			logging.Error("ingestion", "scheduler", fmt.Sprintf("job processing failed: %v", err))
		}
	})

	s.cron.Start()
	logging.Info("ingestion", "scheduler", "ingestion scheduler started")

	return nil
}

// ScheduleSource adds a source to the cron scheduler.
func (s *Scheduler) ScheduleSource(sourceID uint, cronExpr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing schedule for this source if any.
	if existing, ok := s.sources[sourceID]; ok {
		s.cron.Remove(existing.entryID)
		delete(s.sources, sourceID)
		logging.Info("ingestion", "scheduler",
			fmt.Sprintf("removed previous schedule for source %d", sourceID))
	}

	// Validate the cron expression.
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(cronExpr); err != nil {
		// Try without seconds (standard 5-field cron).
		stdParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err2 := stdParser.Parse(cronExpr); err2 != nil {
			return fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
		}
		// Prefix with 0 seconds for the 6-field parser.
		cronExpr = "0 " + cronExpr
	}

	sid := sourceID
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.triggerSourceJob(sid)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule source %d: %w", sourceID, err)
	}

	s.sources[sourceID] = &scheduledSource{
		sourceID: sourceID,
		entryID:  entryID,
		cronExpr: cronExpr,
	}

	logging.Info("ingestion", "scheduler",
		fmt.Sprintf("source %d scheduled with cron '%s' (entry %d)", sourceID, cronExpr, entryID))

	return nil
}

// triggerSourceJob creates an ingestion job for a scheduled source.
func (s *Scheduler) triggerSourceJob(sourceID uint) {
	logging.Info("ingestion", "scheduler", fmt.Sprintf("cron trigger for source %d", sourceID))

	_, err := s.engine.EnqueueJob(EnqueueRequest{
		ImportSourceID: sourceID,
		Priority:       5,
		Mode:           ModeIncremental,
	})
	if err != nil {
		logging.Error("ingestion", "scheduler",
			fmt.Sprintf("failed to enqueue job for source %d: %v", sourceID, err))
	}
}

// HandleMissedRuns handles missed runs after downtime.
// Policy: catch-up-once -- run once to catch up, then resume normal schedule.
func (s *Scheduler) HandleMissedRuns() error {
	logging.Info("ingestion", "scheduler", "checking for missed runs")

	// Find active sources that have a schedule configured.
	var sources []models.ImportSource
	if err := s.db.Where("is_active = ?", true).Find(&sources).Error; err != nil {
		return fmt.Errorf("failed to query active sources: %w", err)
	}

	for _, source := range sources {
		// Check if there are any recent jobs for this source.
		var lastJob models.IngestionJob
		err := s.db.Where("import_source_id = ?", source.ID).
			Order("created_at DESC").
			First(&lastJob).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			logging.Error("ingestion", "scheduler",
				fmt.Sprintf("failed to check last job for source %d: %v", source.ID, err))
			continue
		}

		needsCatchUp := false
		if err == gorm.ErrRecordNotFound {
			// Never been run; needs catch-up.
			needsCatchUp = true
		} else {
			// If last job was more than 24 hours ago, consider it missed.
			if time.Since(lastJob.CreatedAt) > 24*time.Hour {
				needsCatchUp = true
			}
			// If there are incomplete jobs, do not create a new one.
			if lastJob.State == StateRunning || lastJob.State == StateReady || lastJob.State == StateRetrying {
				needsCatchUp = false
			}
		}

		if needsCatchUp {
			logging.Info("ingestion", "scheduler",
				fmt.Sprintf("source %d missed its schedule, enqueuing catch-up job", source.ID))

			_, enqErr := s.engine.EnqueueJob(EnqueueRequest{
				ImportSourceID: source.ID,
				Priority:       3, // Lower priority for catch-up jobs.
				Mode:           ModeIncremental,
			})
			if enqErr != nil {
				logging.Error("ingestion", "scheduler",
					fmt.Sprintf("failed to enqueue catch-up job for source %d: %v", source.ID, enqErr))
			}
		}
	}

	return nil
}

// RemoveSource unschedules a source.
func (s *Scheduler) RemoveSource(sourceID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.sources[sourceID]; ok {
		s.cron.Remove(existing.entryID)
		delete(s.sources, sourceID)
		logging.Info("ingestion", "scheduler",
			fmt.Sprintf("source %d unscheduled", sourceID))
	}
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	logging.Info("ingestion", "scheduler", "stopping ingestion scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	logging.Info("ingestion", "scheduler", "ingestion scheduler stopped")
}
