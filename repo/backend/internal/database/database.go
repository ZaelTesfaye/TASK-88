package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"backend/internal/config"
	"backend/internal/logging"
	"backend/internal/models"
)

var db *gorm.DB

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	logLevel := gormlogger.Warn
	if cfg.AppEnv != "production" {
		logLevel = gormlogger.Info
	}

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(logLevel),
		SkipDefaultTransaction: false,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	logging.Info("database", "connect", "database connection established successfully")

	return db, nil
}

func GetDB() *gorm.DB {
	return db
}

func AutoMigrate(database *gorm.DB) error {
	logging.Info("database", "migrate", "running auto-migrations")

	err := database.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.OrgNode{},
		&models.ContextAssignment{},
		&models.MasterRecord{},
		&models.MasterVersion{},
		&models.MasterVersionItem{},
		&models.DeactivationEvent{},
		&models.ImportSource{},
		&models.IngestionJob{},
		&models.IngestionCheckpoint{},
		&models.IngestionFailure{},
		&models.MediaAsset{},
		&models.AnalyticsKPIDefinition{},
		&models.ReportSchedule{},
		&models.ReportRun{},
		&models.AuditLog{},
		&models.AuditDeleteRequest{},
		&models.SensitiveFieldRegistry{},
		&models.KeyRing{},
		&models.PasswordResetRequest{},
		&models.RetentionPolicy{},
		&models.LegalHold{},
		&models.PurgeRun{},
		&models.IntegrationEndpoint{},
		&models.IntegrationDelivery{},
		&models.ConnectorDefinition{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	logging.Info("database", "migrate", "auto-migrations completed successfully")
	return nil
}

func WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func WithTransactionResult[T any](ctx context.Context, fn func(tx *gorm.DB) (T, error)) (T, error) {
	var result T
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var err error
	result, err = fn(tx)
	if err != nil {
		tx.Rollback()
		return result, err
	}

	if err := tx.Commit().Error; err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}
