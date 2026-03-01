package config

import (
	"os"
	"strings"
	"time"

	"diary-service/internal/pkg/jwt"
)

type AppConfig struct {
	// Server
	HTTPAddr  string
	GRPCAddr  string
	RedisAddr string
	RedisPass string

	// JWT
	JWT jwt.Config

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPFromName string
	SMTPSecure   bool
}

// Load loads environment variables into AppConfig.
func Load() AppConfig {
	return AppConfig{
		HTTPAddr:  getEnv("HTTP_ADDR", ":8001"),
		GRPCAddr:  getEnv("GRPC_ADDR", ":8006"),
		RedisAddr: getEnv("REDIS_ADDR", "redis-cluster:6379"),
		RedisPass: getEnv("REDIS_PASS", ""),

		JWT: jwt.Config{
			PrivPath: getEnv("JWT_PRIVATE_KEY_PATH", "/app/secrets/jwt_private.pem"),
			PubPath:  getEnv("JWT_PUBLIC_KEY_PATH", "/app/secrets/jwt_public.pem"),
			Issuer:   "diary-app",
			Audience: "diary-users",
			TTL:      24 * time.Hour,
			KID:      "diary-key",
		},

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "465"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPass:     getEnv("SMTP_PASS", ""),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "Diary App"),
		SMTPSecure:   strings.ToLower(getEnv("SMTP_SECURE", "true")) == "true",
	}
}

// --- Helper functions ---

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
