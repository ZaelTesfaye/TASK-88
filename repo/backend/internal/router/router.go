package router

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/auth"
	"backend/internal/config"
	appErrors "backend/internal/errors"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/rbac"
)

// SetupRouter configures the Gin engine with all middleware and route groups.
func SetupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.RequestLoggerMiddleware())
	r.Use(appErrors.ErrorHandlerMiddleware())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Correlation-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Correlation-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(middleware.EgressGuardMiddleware(cfg.AllowedHosts))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	api := r.Group("/api/v1")
	registerAllRoutes(api, db)

	return r
}

func registerAllRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	// ---- Public routes (no auth required) ----
	authHandler := handlers.NewAuthHandler(db)
	publicAuth := rg.Group("/auth")
	{
		publicAuth.POST("/login", authHandler.Login)
	}

	// ---- Protected routes (auth required) ----
	protected := rg.Group("")
	protected.Use(auth.AuthRequired(db))

	// Auth routes that require authentication
	protectedAuth := protected.Group("/auth")
	{
		protectedAuth.POST("/logout", authHandler.Logout)
		protectedAuth.POST("/refresh", authHandler.Refresh)
	}

	// Org routes: SystemAdmin only
	orgHandler := handlers.NewOrgHandler(db)
	orgRoutes := protected.Group("/org")
	orgRoutes.Use(rbac.RequireRole(rbac.SystemAdmin))
	{
		orgRoutes.GET("/tree", orgHandler.GetTree)
		orgRoutes.GET("/nodes", orgHandler.ListNodes)
		orgRoutes.POST("/nodes", orgHandler.CreateNode)
		orgRoutes.GET("/nodes/:id", orgHandler.GetNode)
		orgRoutes.PUT("/nodes/:id", orgHandler.UpdateNode)
		orgRoutes.DELETE("/nodes/:id", orgHandler.DeleteNode)
	}
	ctxRoutes := protected.Group("/context")
	ctxRoutes.Use(rbac.RequireRole(rbac.SystemAdmin))
	{
		ctxRoutes.POST("/switch", orgHandler.SwitchContext)
		ctxRoutes.GET("/current", orgHandler.GetCurrentContext)
	}

	// Master routes: permission-based (GET = view, POST/PUT/PATCH = crud)
	masterHandler := handlers.NewMasterHandler(db)
	masterRoutes := protected.Group("/master")
	{
		masterRoutes.GET("/:entity", rbac.RequirePermission("master_data_view"), masterHandler.ListRecords)
		masterRoutes.GET("/:entity/:id", rbac.RequirePermission("master_data_view"), masterHandler.GetRecord)
		masterRoutes.GET("/:entity/:id/history", rbac.RequirePermission("master_data_view"), masterHandler.GetRecordHistory)
		masterRoutes.POST("/:entity", rbac.RequirePermission("master_data_crud"), masterHandler.CreateRecord)
		masterRoutes.PUT("/:entity/:id", rbac.RequirePermission("master_data_crud"), masterHandler.UpdateRecord)
		masterRoutes.POST("/:entity/:id/deactivate", rbac.RequirePermission("master_data_crud"), masterHandler.DeactivateRecord)
	}

	// Version routes: role-based
	versionHandler := handlers.NewVersionHandler(db)
	versionRoutes := protected.Group("/versions")
	{
		versionRoutes.GET("/:entity",
			rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward, rbac.OperationsAnalyst, rbac.StandardUser),
			versionHandler.ListVersions)
		versionRoutes.GET("/:entity/:id",
			rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward, rbac.OperationsAnalyst, rbac.StandardUser),
			versionHandler.GetVersion)
		versionRoutes.GET("/:entity/:id/items",
			rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward, rbac.OperationsAnalyst, rbac.StandardUser),
			versionHandler.ListVersionItems)
		versionRoutes.GET("/:entity/:id/diff",
			rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward, rbac.OperationsAnalyst, rbac.StandardUser),
			versionHandler.DiffVersions)
		versionRoutes.POST("/:entity",
			rbac.RequirePermission("version_draft"),
			versionHandler.CreateVersion)
		versionRoutes.POST("/:entity/:id/review",
			rbac.RequirePermission("version_draft"),
			versionHandler.ReviewVersion)
		versionRoutes.POST("/:entity/:id/items",
			rbac.RequirePermission("version_draft"),
			versionHandler.AddVersionItem)
		versionRoutes.DELETE("/:entity/:id/items/:itemId",
			rbac.RequirePermission("version_draft"),
			versionHandler.RemoveVersionItem)
		versionRoutes.POST("/:entity/:id/activate",
			rbac.RequireRole(rbac.SystemAdmin),
			versionHandler.ActivateVersion)
	}

	// Ingestion routes: SystemAdmin and OperationsAnalyst
	ingestionHandler := handlers.NewIngestionHandler(db)
	ingestionRoutes := protected.Group("/ingestion")
	ingestionRoutes.Use(rbac.RequireRole(rbac.SystemAdmin, rbac.OperationsAnalyst))
	{
		ingestionRoutes.GET("/sources", ingestionHandler.ListSources)
		ingestionRoutes.POST("/sources", ingestionHandler.CreateSource)
		ingestionRoutes.GET("/sources/:id", ingestionHandler.GetSource)
		ingestionRoutes.PUT("/sources/:id", ingestionHandler.UpdateSource)
		ingestionRoutes.DELETE("/sources/:id", ingestionHandler.DeleteSource)
		ingestionRoutes.GET("/jobs", ingestionHandler.ListJobs)
		ingestionRoutes.POST("/jobs", ingestionHandler.CreateJob)
		ingestionRoutes.GET("/jobs/:id", ingestionHandler.GetJob)
		ingestionRoutes.POST("/jobs/:id/retry", ingestionHandler.RetryJob)
		ingestionRoutes.POST("/jobs/:id/acknowledge", ingestionHandler.AcknowledgeJob)
		ingestionRoutes.GET("/jobs/:id/checkpoints", ingestionHandler.ListCheckpoints)
		ingestionRoutes.GET("/jobs/:id/failures", ingestionHandler.ListFailures)
	}

	// Playback routes: read operations open to all authenticated users,
	// mutations (POST/PUT/DELETE) require data_steward or system_admin.
	playbackHandler := handlers.NewPlaybackHandler(db)
	playbackRoutes := protected.Group("/media")
	{
		playbackRoutes.GET("", playbackHandler.ListMedia)
		playbackRoutes.GET("/:id", playbackHandler.GetMedia)
		playbackRoutes.GET("/:id/stream", playbackHandler.StreamAudio)
		playbackRoutes.GET("/:id/cover", playbackHandler.GetCoverArt)
		playbackRoutes.GET("/:id/lyrics/search", playbackHandler.SearchLyrics)
		playbackRoutes.GET("/formats/supported", playbackHandler.GetSupportedFormats)
		// Mutations restricted to elevated roles.
		playbackRoutes.POST("", rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward), playbackHandler.CreateMedia)
		playbackRoutes.PUT("/:id", rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward), playbackHandler.UpdateMedia)
		playbackRoutes.DELETE("/:id", rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward), playbackHandler.DeleteMedia)
		playbackRoutes.POST("/:id/lyrics/parse", rbac.RequireRole(rbac.SystemAdmin, rbac.DataSteward), playbackHandler.ParseLyrics)
	}

	// Analytics routes: OperationsAnalyst and SystemAdmin, scope-enforced
	analyticsHandler := handlers.NewAnalyticsHandler(db)
	analyticsRoutes := protected.Group("/analytics")
	analyticsRoutes.Use(rbac.RequireRole(rbac.SystemAdmin, rbac.OperationsAnalyst))
	analyticsRoutes.Use(rbac.EnforceScopeContext())
	{
		analyticsRoutes.GET("/kpis", analyticsHandler.GetKPIs)
		analyticsRoutes.GET("/kpis/definitions", analyticsHandler.ListKPIDefinitions)
		analyticsRoutes.POST("/kpis/definitions", analyticsHandler.CreateKPIDefinition)
		analyticsRoutes.GET("/kpis/definitions/:code", analyticsHandler.GetKPIDefinition)
		analyticsRoutes.PUT("/kpis/definitions/:code", analyticsHandler.UpdateKPIDefinition)
		analyticsRoutes.DELETE("/kpis/definitions/:code", analyticsHandler.DeleteKPIDefinition)
		analyticsRoutes.GET("/trends", analyticsHandler.GetTrends)
	}

	// Report routes: OperationsAnalyst and SystemAdmin, scope-enforced
	reportHandler := handlers.NewReportHandler(db)
	reportRoutes := protected.Group("/reports")
	reportRoutes.Use(rbac.RequireRole(rbac.SystemAdmin, rbac.OperationsAnalyst))
	reportRoutes.Use(rbac.EnforceScopeContext())
	{
		reportRoutes.GET("/schedules", reportHandler.ListSchedules)
		reportRoutes.POST("/schedules", reportHandler.CreateSchedule)
		reportRoutes.GET("/schedules/:id", reportHandler.GetSchedule)
		reportRoutes.PATCH("/schedules/:id", reportHandler.UpdateSchedule)
		reportRoutes.DELETE("/schedules/:id", reportHandler.DeleteSchedule)
		reportRoutes.POST("/schedules/:id/trigger", reportHandler.TriggerSchedule)
		reportRoutes.GET("/runs", reportHandler.ListRuns)
		reportRoutes.GET("/runs/:id", reportHandler.GetRun)
		reportRoutes.GET("/runs/:id/download", reportHandler.DownloadRun)
		reportRoutes.GET("/runs/:id/access-check", reportHandler.AccessCheck)
	}

	// Audit routes: SystemAdmin only
	auditHandler := handlers.NewAuditHandler(db)
	auditRoutes := protected.Group("/audit")
	auditRoutes.Use(rbac.RequireRole(rbac.SystemAdmin))
	{
		auditRoutes.GET("/logs", auditHandler.ListLogs)
		auditRoutes.GET("/logs/:id", auditHandler.GetLog)
		auditRoutes.GET("/logs/search", auditHandler.SearchLogs)
		auditRoutes.GET("/delete-requests", auditHandler.ListDeleteRequests)
		auditRoutes.POST("/delete-requests", auditHandler.CreateDeleteRequest)
		auditRoutes.GET("/delete-requests/:id", auditHandler.GetDeleteRequest)
		auditRoutes.POST("/delete-requests/:id/approve", auditHandler.ApproveDeleteRequest)
		auditRoutes.POST("/delete-requests/:id/execute", auditHandler.ExecuteDeleteRequest)
	}

	// Security routes: SystemAdmin only
	securityHandler := handlers.NewSecurityHandler(db)
	securityRoutes := protected.Group("/security")
	securityRoutes.Use(rbac.RequireRole(rbac.SystemAdmin))
	{
		securityRoutes.GET("/sensitive-fields", securityHandler.ListSensitiveFields)
		securityRoutes.POST("/sensitive-fields", securityHandler.CreateSensitiveField)
		securityRoutes.PUT("/sensitive-fields/:id", securityHandler.UpdateSensitiveField)
		securityRoutes.DELETE("/sensitive-fields/:id", securityHandler.DeleteSensitiveField)
		securityRoutes.GET("/keys", securityHandler.ListKeys)
		securityRoutes.POST("/keys/rotate", securityHandler.RotateKey)
		securityRoutes.GET("/keys/:id", securityHandler.GetKey)
		securityRoutes.POST("/password-reset", securityHandler.RequestPasswordReset)
		securityRoutes.POST("/password-reset/:id/approve", securityHandler.ApprovePasswordReset)
		securityRoutes.GET("/password-reset", securityHandler.ListPasswordResetRequests)
		securityRoutes.GET("/retention-policies", securityHandler.ListRetentionPolicies)
		securityRoutes.POST("/retention-policies", securityHandler.CreateRetentionPolicy)
		securityRoutes.PUT("/retention-policies/:id", securityHandler.UpdateRetentionPolicy)
		securityRoutes.GET("/legal-holds", securityHandler.ListLegalHolds)
		securityRoutes.POST("/legal-holds", securityHandler.CreateLegalHold)
		securityRoutes.POST("/legal-holds/:id/release", securityHandler.ReleaseLegalHold)
		securityRoutes.POST("/purge-runs/dry-run", securityHandler.DryRunPurge)
		securityRoutes.POST("/purge-runs/execute", securityHandler.ExecutePurge)
		securityRoutes.GET("/purge-runs", securityHandler.ListPurgeRuns)
	}

	// Integration routes: SystemAdmin only
	integrationHandler := handlers.NewIntegrationHandler(db)
	integrationRoutes := protected.Group("/integrations")
	integrationRoutes.Use(rbac.RequireRole(rbac.SystemAdmin))
	{
		integrationRoutes.GET("/endpoints", integrationHandler.ListEndpoints)
		integrationRoutes.POST("/endpoints", integrationHandler.CreateEndpoint)
		integrationRoutes.GET("/endpoints/:id", integrationHandler.GetEndpoint)
		integrationRoutes.PUT("/endpoints/:id", integrationHandler.UpdateEndpoint)
		integrationRoutes.DELETE("/endpoints/:id", integrationHandler.DeleteEndpoint)
		integrationRoutes.POST("/endpoints/:id/test", integrationHandler.TestEndpoint)
		integrationRoutes.GET("/deliveries", integrationHandler.ListDeliveries)
		integrationRoutes.GET("/deliveries/:id", integrationHandler.GetDelivery)
		integrationRoutes.POST("/deliveries/:id/retry", integrationHandler.RetryDelivery)
		integrationRoutes.GET("/connectors", integrationHandler.ListConnectors)
		integrationRoutes.POST("/connectors", integrationHandler.CreateConnector)
		integrationRoutes.GET("/connectors/:id", integrationHandler.GetConnector)
		integrationRoutes.PUT("/connectors/:id", integrationHandler.UpdateConnector)
		integrationRoutes.DELETE("/connectors/:id", integrationHandler.DeleteConnector)
		integrationRoutes.POST("/connectors/:id/health-check", integrationHandler.HealthCheckConnector)
	}
}
