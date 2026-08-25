-- Adds reminder tracking so the pending-order reminder worker sends each
-- customer at most one nudge per order instead of re-notifying on every tick.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMP NULL;

CREATE INDEX IF NOT EXISTS idx_orders_pending_reminder
    ON orders (created_at)
    WHERE status = 'pending' AND reminder_sent_at IS NULL;
