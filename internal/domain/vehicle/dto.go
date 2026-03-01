package vehicle

import "time"

type Category struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	CategoryImage string  `json:"category_image" db:"category_image"`
	Description string    `json:"description,omitempty" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	DisplayOrder int      `json:"display_order" db:"display_order"`
	CreatedBy   int64     `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CategoryCreateRequest is used by handlers/services when creating a new category
type CategoryCreateRequest struct {
	Name          string `json:"name" binding:"required"`
	CategoryImage string `json:"category_image" binding:"required"`
	Description   string `json:"description,omitempty"`
	DisplayOrder  int    `json:"display_order,omitempty"`
	IsActive      bool   `json:"is_active,omitempty"`
}

// CategoryUpdateRequest is used to update existing category info
type CategoryUpdateRequest struct {
	Name          *string `json:"name,omitempty"`
	CategoryImage *string `json:"category_image,omitempty"`
	Description   *string `json:"description,omitempty"`
	DisplayOrder  *int    `json:"display_order,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

// CategoryVehicle links a vehicle to a category (many-to-many)
type CategoryVehicle struct {
	ID         int64     `json:"id" db:"id"`
	CategoryID int64     `json:"category_id" db:"category_id"`
	VehicleID  int64     `json:"vehicle_id" db:"vehicle_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// CategoryInfo for lightweight responses (list views)
type CategoryInfo struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	CategoryImage string    `json:"category_image"`
	IsActive      bool      `json:"is_active"`
	DisplayOrder  int       `json:"display_order"`
	CreatedAt     time.Time `json:"created_at"`
}