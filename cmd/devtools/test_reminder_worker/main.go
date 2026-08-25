// Temporary debug tool: runs the real pending-order reminder worker's
// RunOnce against the live production database and mailer, to confirm the
// query/eligibility logic behaves correctly against real data. Any state
// mutated (reminder_sent_at) is reported so it can be reset afterward.
// Remove after debugging.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"zentora-service/internal/config"
	"zentora-service/internal/db"
	"zentora-service/internal/repository/postgres"
	"zentora-service/internal/service/email"
	workerservice "zentora-service/internal/service/worker"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on system env vars")
	}
	cfg := config.Load()

	pool, err := db.ConnectDB()
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	orderRepo := postgres.NewOrderRepository(pool)

	ctx := context.Background()
	candidates, err := orderRepo.ListPendingOrdersNeedingReminder(ctx, cfg.OrderReminderPendingAfter)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	fmt.Printf("ListPendingOrdersNeedingReminder(olderThan=%s) => %d eligible candidate(s)\n", cfg.OrderReminderPendingAfter, len(candidates))
	for _, c := range candidates {
		fmt.Printf("  - order=%s email=%s total=%s %.2f created=%s\n", c.OrderNumber, c.CustomerEmail, c.Currency, c.TotalAmount, c.CreatedAt)
	}

	emailSender := email.NewEmailSender(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass,
		cfg.SMTPFromName, cfg.BaseURL, cfg.LogoURL, cfg.SMTPSecure,
	)
	orderMailer := email.NewOrderEmailSender(emailSender, cfg.AdminEmail, cfg.StoreBaseURL)
	reminderJob := workerservice.NewOrderReminderJobService(
		orderRepo, orderMailer, cfg.OrderReminderPendingAfter, cfg.WorkerOrderReminderInterval, log.Default(),
	)

	fmt.Println("Running real reminderJob.RunOnce() against production...")
	if err := reminderJob.RunOnce(ctx); err != nil {
		log.Fatalf("RunOnce failed: %v", err)
	}
	fmt.Println("RunOnce complete (see logs above for any sends/marks).")

	// Preview: build a realistic reminder using the oldest real pending order's
	// actual data, but send it to the debug inbox instead of the (nonexistent)
	// guest email, purely so the requester can see real content end-to-end.
	const previewQ = `SELECT order_number, total_amount, currency, created_at FROM orders WHERE status='pending' ORDER BY created_at ASC LIMIT 1`
	var orderNumber, currency string
	var totalAmount float64
	var createdAt time.Time
	if err := pool.QueryRow(ctx, previewQ).Scan(&orderNumber, &totalAmount, &currency, &createdAt); err == nil {
		fmt.Printf("Sending live-data PREVIEW reminder (real order %s) to %s...\n", orderNumber, cfg.AdminEmail)
		if err := orderMailer.SendPendingOrderReminder(cfg.AdminEmail, "Preview", orderNumber, totalAmount, currency, createdAt); err != nil {
			log.Fatalf("preview send failed: %v", err)
		}
		fmt.Println("✅ preview reminder sent")
	}
}
