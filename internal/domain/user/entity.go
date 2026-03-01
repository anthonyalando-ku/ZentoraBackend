// internal/domain/user/entity.go
package user

import "time"

// Address represents a saved shipping/billing address for a user.
type Address struct {
	ID           int64     `json:"id" db:"id"`
	UserID       int64     `json:"user_id" db:"user_id"`
	FullName     string    `json:"full_name" db:"full_name"`
	PhoneNumber  string    `json:"phone_number" db:"phone_number"`
	Country      string    `json:"country" db:"country"`
	County       *string   `json:"county" db:"county"`
	City         string    `json:"city" db:"city"`
	Area         *string   `json:"area" db:"area"`
	PostalCode   *string   `json:"postal_code" db:"postal_code"`
	AddressLine1 string    `json:"address_line_1" db:"address_line_1"`
	AddressLine2 *string   `json:"address_line_2" db:"address_line_2"`
	IsDefault    bool      `json:"is_default" db:"is_default"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
