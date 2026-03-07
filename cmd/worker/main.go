package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"zentora-service/internal/config"
	"zentora-service/internal/db"
	"zentora-service/internal/repository/postgres"
	workerservice "zentora-service/internal/service/worker"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[WORKER] No .env file found, relying on system env vars")
	}

	cfg := config.Load()

	pool, err := db.ConnectDB()
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	repo := postgres.NewDiscoveryRepository(pool)
	jobService := workerservice.NewMetricsJobService(repo, cfg.WorkerMetricsInterval, log.Default())

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	log.Printf("🚀 Metrics worker running with interval %s", cfg.WorkerMetricsInterval)
	if err := jobService.Start(ctx); err != nil {
		log.Fatalf("❌ Metrics worker stopped with error: %v", err)
	}
	log.Println("✅ Metrics worker stopped gracefully")
}
