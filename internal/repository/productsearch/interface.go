package productsearchrepo

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Repository interface {
	// UpsertForProductTx builds and stores search doc + tsvector.
	// MUST be called within the same tx as product creation/update.
	UpsertForProductTx(ctx context.Context, tx pgx.Tx, productID int64, searchDocument string) error
}