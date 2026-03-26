package postgres

import (
	"context"
	"errors"
	"fmt"

	"zentora-service/internal/domain/homepage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HomepageRepository provides scalable read/write access to homepage_sections.
//
// Read path design for high-throughput:
//   - ListActiveSections is the hot read path — one query, ordered, no joins.
//     The service layer resolves products in parallel from the catalog repos.
//   - The table is small (O(tens) of rows) so a covering index on
//     (is_active, sort_order) keeps every read sub-millisecond at any scale.
//   - The service layer caches the fully assembled HomepageResponse in Redis;
//     this repo is only hit on cache misses or admin mutations.
type HomepageRepository struct {
	db *pgxpool.Pool
}

func NewHomepageRepository(db *pgxpool.Pool) *HomepageRepository {
	return &HomepageRepository{db: db}
}

// ─── Reads ────────────────────────────────────────────────────────────────────

// ListActiveSections returns all active sections ordered by sort_order.
// This is the single hot-path query for the public homepage.
func (r *HomepageRepository) ListActiveSections(ctx context.Context) ([]homepage.Section, error) {
	const q = `
		SELECT id, title, type, reference_id, sort_order, is_active
		FROM   homepage_sections
		WHERE  is_active = TRUE
		ORDER  BY sort_order ASC, id ASC`

	return r.queryList(ctx, q)
}

// ListSections returns sections with optional filtering (admin use).
func (r *HomepageRepository) ListSections(ctx context.Context, f homepage.ListFilter) ([]homepage.Section, error) {
	q := `
		SELECT id, title, type, reference_id, sort_order, is_active
		FROM   homepage_sections
		WHERE  1=1`

	args := make([]any, 0, 2)
	idx := 1

	if f.ActiveOnly {
		q += fmt.Sprintf(" AND is_active = $%d", idx)
		args = append(args, true)
		idx++
	}
	if f.Type != nil {
		q += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, string(*f.Type))
		idx++
	}
	_ = idx
	q += " ORDER BY sort_order ASC, id ASC"

	return r.queryListArgs(ctx, q, args...)
}

// GetSectionByID returns a single section by primary key.
func (r *HomepageRepository) GetSectionByID(ctx context.Context, id int64) (*homepage.Section, error) {
	const q = `
		SELECT id, title, type, reference_id, sort_order, is_active
		FROM   homepage_sections
		WHERE  id = $1`

	row := r.db.QueryRow(ctx, q, id)
	s, err := scanSection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, homepage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get homepage section: %w", err)
	}
	return s, nil
}

// ─── Writes ───────────────────────────────────────────────────────────────────

// CreateSection inserts a new section and populates ID on the struct.
func (r *HomepageRepository) CreateSection(ctx context.Context, s *homepage.Section) error {
	const q = `
		INSERT INTO homepage_sections (title, type, reference_id, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	err := r.db.QueryRow(ctx, q,
		s.Title, string(s.Type), s.ReferenceID, s.SortOrder, s.IsActive,
	).Scan(&s.ID)
	if err != nil {
		return fmt.Errorf("create homepage section: %w", err)
	}
	return nil
}

// UpdateSection applies a full update to an existing section.
func (r *HomepageRepository) UpdateSection(ctx context.Context, s *homepage.Section) error {
	const q = `
		UPDATE homepage_sections
		SET    title        = $1,
		       type         = $2,
		       reference_id = $3,
		       sort_order   = $4,
		       is_active    = $5
		WHERE  id = $6`

	result, err := r.db.Exec(ctx, q,
		s.Title, string(s.Type), s.ReferenceID, s.SortOrder, s.IsActive, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update homepage section: %w", err)
	}
	if result.RowsAffected() == 0 {
		return homepage.ErrNotFound
	}
	return nil
}

// DeleteSection hard-deletes a section by ID.
func (r *HomepageRepository) DeleteSection(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM homepage_sections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete homepage section: %w", err)
	}
	if result.RowsAffected() == 0 {
		return homepage.ErrNotFound
	}
	return nil
}

// ReorderSections replaces all sort_order values atomically in a single
// transaction. Any section IDs not present in the list are left unchanged.
func (r *HomepageRepository) ReorderSections(ctx context.Context, items []homepage.ReorderItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reorder transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, item := range items {
		result, err := tx.Exec(ctx,
			`UPDATE homepage_sections SET sort_order = $1 WHERE id = $2`,
			item.SortOrder, item.ID,
		)
		if err != nil {
			return fmt.Errorf("reorder section %d: %w", item.ID, err)
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("section %d: %w", item.ID, homepage.ErrNotFound)
		}
	}
	return tx.Commit(ctx)
}

// ToggleActive sets is_active for a section without touching other fields.
func (r *HomepageRepository) ToggleActive(ctx context.Context, id int64, active bool) error {
	result, err := r.db.Exec(ctx,
		`UPDATE homepage_sections SET is_active = $1 WHERE id = $2`,
		active, id,
	)
	if err != nil {
		return fmt.Errorf("toggle homepage section: %w", err)
	}
	if result.RowsAffected() == 0 {
		return homepage.ErrNotFound
	}
	return nil
}

// ─── Private helpers ──────────────────────────────────────────────────────────

func (r *HomepageRepository) queryList(ctx context.Context, q string, args ...any) ([]homepage.Section, error) {
	return r.queryListArgs(ctx, q, args...)
}

func (r *HomepageRepository) queryListArgs(ctx context.Context, q string, args ...any) ([]homepage.Section, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list homepage sections: %w", err)
	}
	defer rows.Close()

	var out []homepage.Section
	for rows.Next() {
		s, err := scanSectionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan homepage section: %w", err)
		}
		out = append(out, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate homepage sections: %w", err)
	}
	return out, nil
}

// scanSection scans a pgx.Row (single-row query).
func scanSection(row pgx.Row) (*homepage.Section, error) {
	var s homepage.Section
	var t string
	if err := row.Scan(&s.ID, &s.Title, &t, &s.ReferenceID, &s.SortOrder, &s.IsActive); err != nil {
		return nil, err
	}
	s.Type = homepage.SectionType(t)
	return &s, nil
}

// scanSectionRow scans a pgx.Rows (multi-row query).
func scanSectionRow(rows pgx.Rows) (*homepage.Section, error) {
	var s homepage.Section
	var t string
	if err := rows.Scan(&s.ID, &s.Title, &t, &s.ReferenceID, &s.SortOrder, &s.IsActive); err != nil {
		return nil, err
	}
	s.Type = homepage.SectionType(t)
	return &s, nil
}