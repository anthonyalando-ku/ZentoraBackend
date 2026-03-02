package brand

import (
	"database/sql"
	"time"
)

type Brand struct {
	ID        int64          `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	LogoURL   sql.NullString `json:"logo_url"`
	IsActive  bool           `json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
}