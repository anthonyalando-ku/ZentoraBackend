package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"

	"zentora-service/internal/config"
	"zentora-service/internal/db"
	"zentora-service/internal/repository/postgres"
	"zentora-service/internal/service/email"
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

	discoveryRepo := postgres.NewDiscoveryRepository(pool)
	metricsJob := workerservice.NewMetricsJobService(discoveryRepo, cfg.WorkerMetricsInterval, log.Default())

	emailSender := email.NewEmailSender(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass,
		cfg.SMTPFromName, cfg.BaseURL, cfg.LogoURL, cfg.SMTPSecure,
	)
	orderMailer := email.NewOrderEmailSender(emailSender, cfg.AdminEmail)
	orderRepo := postgres.NewOrderRepository(pool)
	reminderJob := workerservice.NewOrderReminderJobService(
		orderRepo, orderMailer, cfg.OrderReminderPendingAfter, cfg.OrderReminderRenotifyAfter, cfg.WorkerOrderReminderInterval, log.Default(),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("🚀 Metrics worker running with interval %s", cfg.WorkerMetricsInterval)
		if err := metricsJob.Start(ctx); err != nil {
			log.Printf("❌ Metrics worker stopped with error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("🚀 Pending-order reminder worker running with interval %s (reminds after %s pending)",
			cfg.WorkerOrderReminderInterval, cfg.OrderReminderPendingAfter)
		if err := reminderJob.Start(ctx); err != nil {
			log.Printf("❌ Pending-order reminder worker stopped with error: %v", err)
		}
	}()

	wg.Wait()
	log.Println("✅ Workers stopped gracefully")
}
