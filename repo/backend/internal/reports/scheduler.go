package reports

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"backend/internal/logging"
	"backend/internal/models"
)

// ReportScheduler manages cron-based report generation.
type ReportScheduler struct {
	db       *gorm.DB
	service  *ReportService
	cron     *cron.Cron
	timezone string
}

// NewReportScheduler creates a new ReportScheduler.
func NewReportScheduler(db *gorm.DB, service *ReportService, timezone string) *ReportScheduler {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		logging.Warn("reports", "scheduler", fmt.Sprintf("invalid timezone %q, falling back to UTC", timezone))
		loc = time.UTC
	}

	return &ReportScheduler{
		db:       db,
		service:  service,
		cron:     cron.New(cron.WithLocation(loc), cron.WithSeconds()),
		timezone: timezone,
	}
}

// Start loads all enabled schedules from the database and starts the cron scheduler.
func (s *ReportScheduler) Start() error {
	var schedules []models.ReportSchedule
	if err := s.db.Where("is_active = ?", true).Find(&schedules).Error; err != nil {
		return fmt.Errorf("failed to load active schedules: %w", err)
	}

	for _, schedule := range schedules {
		sched := schedule // capture loop variable
		_, err := s.cron.AddFunc(sched.CronExpr, func() {
			s.runSchedule(sched.ID)
		})
		if err != nil {
			logging.Error("reports", "scheduler",
				fmt.Sprintf("failed to add cron entry for schedule %d (%s): %v", sched.ID, sched.Name, err))
			continue
		}
		logging.Info("reports", "scheduler",
			fmt.Sprintf("registered schedule %d (%s) with cron %q", sched.ID, sched.Name, sched.CronExpr))
	}

	s.cron.Start()
	logging.Info("reports", "scheduler", fmt.Sprintf("started with %d active schedules", len(schedules)))
	return nil
}

// runSchedule creates and executes a report run for the given schedule.
func (s *ReportScheduler) runSchedule(scheduleID uint) {
	logging.Info("reports", "scheduler", fmt.Sprintf("triggering scheduled report for schedule %d", scheduleID))

	// Use 0 for requestedBy to indicate system-triggered
	run, err := s.service.ExecuteReport(scheduleID, 0)
	if err != nil {
		logging.Error("reports", "scheduler",
			fmt.Sprintf("failed to execute report for schedule %d: %v", scheduleID, err))
		return
	}

	if run.State == "failed" {
		logging.Warn("reports", "scheduler",
			fmt.Sprintf("report run %d for schedule %d finished with state: failed (%s)",
				run.ID, scheduleID, run.FailureReason))
	} else {
		logging.Info("reports", "scheduler",
			fmt.Sprintf("report run %d for schedule %d completed successfully (%d rows)",
				run.ID, scheduleID, run.RowCount))
	}
}

// HandleMissedRuns checks for missed runs and handles them per the specified policy.
// policy "catch-up-once" = run once to catch up for each schedule that missed a run.
// policy "skip" = mark missed runs as skipped without executing.
func (s *ReportScheduler) HandleMissedRuns(policy string) error {
	var schedules []models.ReportSchedule
	if err := s.db.Where("is_active = ?", true).Find(&schedules).Error; err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	for _, schedule := range schedules {
		// Find the most recent run for this schedule
		var lastRun models.ReportRun
		result := s.db.Where("schedule_id = ?", schedule.ID).Order("created_at DESC").First(&lastRun)

		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			logging.Error("reports", "missed_runs",
				fmt.Sprintf("failed to check last run for schedule %d: %v", schedule.ID, result.Error))
			continue
		}

		// Determine the expected next-run time based on cron expression
		parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		cronSchedule, err := parser.Parse(schedule.CronExpr)
		if err != nil {
			logging.Error("reports", "missed_runs",
				fmt.Sprintf("failed to parse cron for schedule %d: %v", schedule.ID, err))
			continue
		}

		var lastRunTime time.Time
		if result.Error == gorm.ErrRecordNotFound {
			lastRunTime = schedule.CreatedAt
		} else {
			lastRunTime = lastRun.CreatedAt
		}

		nextExpected := cronSchedule.Next(lastRunTime)
		now := time.Now()

		if nextExpected.Before(now) {
			// A run was missed
			switch policy {
			case "catch-up-once":
				logging.Info("reports", "missed_runs",
					fmt.Sprintf("catching up schedule %d (missed at %s)", schedule.ID, nextExpected.Format(time.RFC3339)))
				go s.runSchedule(schedule.ID)

			case "skip":
				logging.Info("reports", "missed_runs",
					fmt.Sprintf("skipping missed run for schedule %d (was due at %s)", schedule.ID, nextExpected.Format(time.RFC3339)))
				// Create a skipped run record
				skippedAt := time.Now()
				skippedRun := models.ReportRun{
					ScheduleID:    schedule.ID,
					State:         "skipped",
					FailureReason: fmt.Sprintf("missed run at %s, skipped per policy", nextExpected.Format(time.RFC3339)),
					StartedAt:     &skippedAt,
					CompletedAt:   &skippedAt,
				}
				s.db.Create(&skippedRun)

			default:
				logging.Warn("reports", "missed_runs",
					fmt.Sprintf("unknown missed run policy %q, skipping", policy))
			}
		}
	}

	return nil
}

// Stop gracefully stops the cron scheduler and waits for running jobs to finish.
func (s *ReportScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	logging.Info("reports", "scheduler", "stopped")
}
