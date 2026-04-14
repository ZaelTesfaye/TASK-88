package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/ingestion"
	"backend/internal/logging"
	"backend/internal/reports"
	"backend/internal/router"
)

func main() {
	cfg := config.LoadConfig()

	logging.InitLogger(cfg)
	defer logging.Sync()

	logging.Info("server", "startup", "initializing Multi-Org Data & Media Operations Hub")
	logging.Info("server", "startup", fmt.Sprintf("environment: %s, timezone: %s", cfg.AppEnv, cfg.AppTimezone))

	db, err := database.Connect(cfg)
	if err != nil {
		logging.Error("server", "startup", fmt.Sprintf("database connection failed: %v", err))
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logging.Error("server", "startup", fmt.Sprintf("failed to get underlying sql.DB: %v", err))
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := database.AutoMigrate(db); err != nil {
		logging.Error("server", "startup", fmt.Sprintf("auto-migration failed: %v", err))
		os.Exit(1)
	}

	// Initialize and start ingestion scheduler
	jobEngine := ingestion.NewJobEngine(db)
	ingestionScheduler := ingestion.NewScheduler(db, jobEngine)
	if err := ingestionScheduler.Start(); err != nil {
		logging.Error("server", "startup", fmt.Sprintf("ingestion scheduler failed to start: %v", err))
		os.Exit(1)
	}
	logging.Info("server", "startup", "ingestion scheduler started")

	// Initialize and start report scheduler
	reportService := reports.NewReportService(db)
	reportScheduler := reports.NewReportScheduler(db, reportService, cfg.AppTimezone)
	if err := reportScheduler.Start(); err != nil {
		logging.Error("server", "startup", fmt.Sprintf("report scheduler failed to start: %v", err))
		os.Exit(1)
	}
	// Handle missed report runs per configured policy
	if err := reportScheduler.HandleMissedRuns(cfg.MissedRunPolicy); err != nil {
		logging.Error("server", "startup", fmt.Sprintf("report missed runs handling failed: %v", err))
	}
	logging.Info("server", "startup", "report scheduler started")

	r := router.SetupRouter(cfg, db)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logging.Info("server", "startup", fmt.Sprintf("egress guard active, allowed hosts: %v", cfg.AllowedHosts))

	if cfg.EnableTLS {
		if cfg.TLSCertPath == "" || cfg.TLSKeyPath == "" {
			logging.Error("server", "startup", "ENABLE_TLS=true but TLS_CERT_PATH or TLS_KEY_PATH is empty")
			os.Exit(1)
		}
		logging.Info("server", "startup", fmt.Sprintf("starting HTTPS server on %s (cert=%s, key=%s)", addr, cfg.TLSCertPath, cfg.TLSKeyPath))
		go func() {
			if err := srv.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); err != nil && err != http.ErrServerClosed {
				logging.Error("server", "listen", fmt.Sprintf("HTTPS server error: %v", err))
				os.Exit(1)
			}
		}()
	} else {
		logging.Info("server", "startup", fmt.Sprintf("starting HTTP server on %s", addr))
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logging.Error("server", "listen", fmt.Sprintf("HTTP server error: %v", err))
				os.Exit(1)
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logging.Info("server", "shutdown", fmt.Sprintf("received signal %v, initiating graceful shutdown", sig))

	// Stop schedulers before shutting down the HTTP server
	ingestionScheduler.Stop()
	logging.Info("server", "shutdown", "ingestion scheduler stopped")
	reportScheduler.Stop()
	logging.Info("server", "shutdown", "report scheduler stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logging.Error("server", "shutdown", fmt.Sprintf("forced shutdown: %v", err))
		os.Exit(1)
	}

	logging.Info("server", "shutdown", "server stopped gracefully")
}
