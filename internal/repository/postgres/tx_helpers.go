// internal/repository/postgres/tx_helpers.go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// withTx executes fn inside a pgx transaction.  The transaction is committed
// if fn returns nil, and rolled back (silently) otherwise.
func withTx(ctx context.Context, tx pgx.Tx, fn func() error) error {
	if err := fn(); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}