package admin
// CreateAdminRequest represents the request for creating a new admin
type CreateAdminRequest struct {
	FullName  string `json:"full_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Phone     string `json:"phone" validate:"required"`
	Password  string `json:"password" validate:"required,min=8"`
	CreatedBy *int64  `json:"created_by" validate:"required"`
}

// UpdateAdminRequest represents the request for updating an admin
type UpdateAdminRequest struct {
	ID       int64   `json:"id" validate:"required"`
	FullName string  `json:"full_name,omitempty"`
	Email    string  `json:"email,omitempty" validate:"omitempty,email"`
	Phone    string  `json:"phone,omitempty"`
	Password string  `json:"password,omitempty" validate:"omitempty,min=8"`
	IsActive *bool   `json:"is_active,omitempty"`
}