// internal/repository/postgres/user_address_repo.go
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"diary-service/internal/domain/user"
	xerrors "diary-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserAddressRepository implements user.AddressRepository using pgxpool.
type UserAddressRepository struct {
	db *pgxpool.Pool
}

// NewUserAddressRepository creates a new UserAddressRepository.
func NewUserAddressRepository(db *pgxpool.Pool) *UserAddressRepository {
	return &UserAddressRepository{db: db}
}

// CreateAddress inserts a new address for a user.
func (r *UserAddressRepository) CreateAddress(ctx context.Context, a *user.Address) error {
	query := `
		INSERT INTO user_addresses
			(user_id, full_name, phone_number, country, county, city, area,
			 postal_code, address_line_1, address_line_2, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query,
		a.UserID, a.FullName, a.PhoneNumber, a.Country, a.County,
		a.City, a.Area, a.PostalCode, a.AddressLine1, a.AddressLine2,
		a.IsDefault,
	).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create address: %w", err)
	}
	return nil
}

// GetAddressByID retrieves a single address by its primary key.
func (r *UserAddressRepository) GetAddressByID(ctx context.Context, id int64) (*user.Address, error) {
	query := `
		SELECT id, user_id, full_name, phone_number, country, county, city, area,
		       postal_code, address_line_1, address_line_2, is_default, created_at
		FROM user_addresses
		WHERE id = $1
	`
	var a user.Address
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.UserID, &a.FullName, &a.PhoneNumber, &a.Country, &a.County,
		&a.City, &a.Area, &a.PostalCode, &a.AddressLine1, &a.AddressLine2,
		&a.IsDefault, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}
	return &a, nil
}

// ListAddressesByUser returns all addresses belonging to a user.
func (r *UserAddressRepository) ListAddressesByUser(ctx context.Context, userID int64) ([]user.Address, error) {
	query := `
		SELECT id, user_id, full_name, phone_number, country, county, city, area,
		       postal_code, address_line_1, address_line_2, is_default, created_at
		FROM user_addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses: %w", err)
	}
	defer rows.Close()

	var addresses []user.Address
	for rows.Next() {
		var a user.Address
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.FullName, &a.PhoneNumber, &a.Country, &a.County,
			&a.City, &a.Area, &a.PostalCode, &a.AddressLine1, &a.AddressLine2,
			&a.IsDefault, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan address: %w", err)
		}
		addresses = append(addresses, a)
	}
	return addresses, nil
}

// UpdateAddress replaces all mutable fields of an address.
func (r *UserAddressRepository) UpdateAddress(ctx context.Context, a *user.Address) error {
	query := `
		UPDATE user_addresses
		SET full_name      = $1,
		    phone_number   = $2,
		    country        = $3,
		    county         = $4,
		    city           = $5,
		    area           = $6,
		    postal_code    = $7,
		    address_line_1 = $8,
		    address_line_2 = $9,
		    is_default     = $10
		WHERE id = $11
	`
	result, err := r.db.Exec(ctx, query,
		a.FullName, a.PhoneNumber, a.Country, a.County,
		a.City, a.Area, a.PostalCode, a.AddressLine1, a.AddressLine2,
		a.IsDefault, a.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update address: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

// DeleteAddress removes an address by ID.
func (r *UserAddressRepository) DeleteAddress(ctx context.Context, id int64) error {
	query := `DELETE FROM user_addresses WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}
	return nil
}

// SetDefaultAddress marks one address as default and clears the flag on all
// others owned by the same user.
func (r *UserAddressRepository) SetDefaultAddress(ctx context.Context, userID, addressID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Clear existing default
	if _, err := tx.Exec(ctx,
		`UPDATE user_addresses SET is_default = FALSE WHERE user_id = $1`,
		userID,
	); err != nil {
		return fmt.Errorf("failed to clear default addresses: %w", err)
	}

	// Set new default
	result, err := tx.Exec(ctx,
		`UPDATE user_addresses SET is_default = TRUE WHERE id = $1 AND user_id = $2`,
		addressID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to set default address: %w", err)
	}
	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return tx.Commit(ctx)
}
