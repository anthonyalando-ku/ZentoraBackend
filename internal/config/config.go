package config

import (
	"os"
	"strings"
	"time"

	"zentora-service/internal/pkg/jwt"
)

type AppConfig struct {
	// Server
	HTTPAddr              string
	GRPCAddr              string
	RedisAddr             string
	RedisPass             string
	WorkerMetricsInterval time.Duration

	// JWT
	JWT jwt.Config

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPFromName string
	SMTPSecure   bool

	BaseURL string
	LogoURL string

	ImageKitPrivateKey string
	ImageKitPublicKey  string
	ImageKitURL        string
}

// Load loads environment variables into AppConfig.
func Load() AppConfig {
	return AppConfig{
		HTTPAddr:              getEnv("HTTP_ADDR", ":8002"),
		GRPCAddr:              getEnv("GRPC_ADDR", ":8006"),
		RedisAddr:             getEnv("REDIS_ADDR", "redis-zentora:6379"),
		RedisPass:             getEnv("REDIS_PASS", ""),
		WorkerMetricsInterval: getEnvDuration("WORKER_METRICS_INTERVAL", 15*time.Minute),
		ImageKitPrivateKey:    getEnv("IMAGE_KIT_PRIVATE_KEY", ""),
		ImageKitPublicKey:     getEnv("IMAGE_KIT_PUBLIC_KEY", ""),
		ImageKitURL:           getEnv("IMAGE_KIT_URL_ENDPOINT", ""),

		JWT: jwt.Config{
			PrivPath: getEnv("JWT_PRIVATE_KEY_PATH", "/app/secrets/jwt_private.pem"),
			PubPath:  getEnv("JWT_PUBLIC_KEY_PATH", "/app/secrets/jwt_public.pem"),
			Issuer:   "zentora-app",
			Audience: "zentora-users",
			TTL:      24 * time.Hour,
			KID:      "zentora-key",
		},

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "465"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPass:     getEnv("SMTP_PASS", ""),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "Zentora Support"),
		SMTPSecure:   strings.ToLower(getEnv("SMTP_SECURE", "true")) == "true",
		BaseURL:      getEnv("BASE_URL", ""),
		LogoURL:      getEnv("LOGO_URL", "https://ik.imagekit.io/anthonyalando/zentora/zentora_logo_clear.png"),
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

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}
