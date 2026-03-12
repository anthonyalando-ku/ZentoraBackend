package postgres

import (
	"context"
	"fmt"
	"strings"

	productsearchrepo "zentora-service/internal/repository/productsearch"

	"github.com/jackc/pgx/v5"
)

type ProductSearchRepository struct{}

func NewProductSearchRepository() *ProductSearchRepository {
	return &ProductSearchRepository{}
}

var _ productsearchrepo.Repository = (*ProductSearchRepository)(nil)

// Uses built-in Postgres full text search.
// Simple config: 'english'. If you want KES/local language, we can make this configurable.
func (r *ProductSearchRepository) UpsertForProductTx(ctx context.Context, tx pgx.Tx, productID int64, searchDocument string) error {
	if tx == nil {
		return fmt.Errorf("tx is required")
	}
	if productID <= 0 {
		return fmt.Errorf("invalid productID")
	}
	searchDocument = strings.TrimSpace(searchDocument)
	if searchDocument == "" {
		return fmt.Errorf("searchDocument is required")
	}

	const q = `
		INSERT INTO product_search_documents (product_id, search_document, search_vector, updated_at)
		VALUES ($1, $2, to_tsvector('english', $2), NOW())
		ON CONFLICT (product_id) DO UPDATE SET
			search_document = EXCLUDED.search_document,
			search_vector   = EXCLUDED.search_vector,
			updated_at      = NOW()
	`
	_, err := tx.Exec(ctx, q, productID, searchDocument)
	if err != nil {
		return fmt.Errorf("upsert product_search_document: %w", err)
	}
	return nil
}