package config

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret string
	JWTIssuer string

	AppPort    string
	AppTimezone string
	AppEnv     string

	EnableTLS       bool
	TLSCertPath     string
	TLSKeyPath      string
	EnableBiometric bool

	AllowedHosts []string
	CORSOrigins  []string
	LogLevel     string

	MasterKeyHex         string
	KeyRotationDays      int
	RetentionPurgeDays   int
	MissedRunPolicy      string
	DirSyncIntervalMins  int
}

var (
	instance *Config
	once     sync.Once
)

func LoadConfig() *Config {
	once.Do(func() {
		_ = godotenv.Load()

		instance = &Config{
			DBHost:     envOrDefault("DB_HOST", "127.0.0.1"),
			DBPort:     envOrDefault("DB_PORT", "3306"),
			DBUser:     envOrDefault("DB_USER", "root"),
			DBPassword: envOrDefault("DB_PASSWORD", ""),
			DBName:     envOrDefault("DB_NAME", "multi_org_hub"),

			JWTSecret: envOrDefault("JWT_SECRET", "change-me-in-production"),
			JWTIssuer: envOrDefault("JWT_ISSUER", "multi-org-hub"),

			AppPort:    envOrDefault("APP_PORT", "8080"),
			AppTimezone: envOrDefault("APP_TIMEZONE", "America/New_York"),
			AppEnv:     envOrDefault("APP_ENV", "production"),

			EnableTLS:       envOrDefaultBool("ENABLE_TLS", false),
			TLSCertPath:    envOrDefault("TLS_CERT_PATH", ""),
			TLSKeyPath:     envOrDefault("TLS_KEY_PATH", ""),
			EnableBiometric: envOrDefaultBool("ENABLE_BIOMETRIC", false),

			AllowedHosts: parseAllowedHosts(envOrDefault("ALLOWED_HOSTS", "127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")),
			CORSOrigins:  parseCORSOrigins(envOrDefault("CORS_ORIGINS", "http://localhost:3000")),
			LogLevel:     envOrDefault("LOG_LEVEL", "info"),

			MasterKeyHex:        envOrDefault("MASTER_KEY_HEX", ""),
			KeyRotationDays:     envOrDefaultInt("KEY_ROTATION_DAYS", 90),
			RetentionPurgeDays:  envOrDefaultInt("RETENTION_PURGE_DAYS", 30),
			MissedRunPolicy:     envOrDefault("MISSED_RUN_POLICY", "catch-up-once"),
			DirSyncIntervalMins: envOrDefaultInt("DIR_SYNC_INTERVAL_MINUTES", 15),
		}
	})
	return instance
}

func GetConfig() *Config {
	if instance == nil {
		return LoadConfig()
	}
	return instance
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func envOrDefaultBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func envOrDefaultInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseAllowedHosts(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
