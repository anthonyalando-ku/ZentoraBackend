// internal/domain/user/dto.go
package user

type CreateAddressRequest struct {
	FullName     string  `json:"full_name" binding:"required,max=255"`
	PhoneNumber  string  `json:"phone_number" binding:"required,max=30"`
	Country      string  `json:"country" binding:"required,max=100"`
	County       *string `json:"county"`
	City         string  `json:"city" binding:"required,max=100"`
	Area         *string `json:"area"`
	PostalCode   *string `json:"postal_code"`
	AddressLine1 string  `json:"address_line_1" binding:"required,max=255"`
	AddressLine2 *string `json:"address_line_2"`
	IsDefault    bool    `json:"is_default"`
}

type UpdateAddressRequest struct {
	FullName     *string `json:"full_name" binding:"omitempty,max=255"`
	PhoneNumber  *string `json:"phone_number" binding:"omitempty,max=30"`
	Country      *string `json:"country" binding:"omitempty,max=100"`
	County       *string `json:"county"`
	City         *string `json:"city" binding:"omitempty,max=100"`
	Area         *string `json:"area"`
	PostalCode   *string `json:"postal_code"`
	AddressLine1 *string `json:"address_line_1" binding:"omitempty,max=255"`
	AddressLine2 *string `json:"address_line_2"`
	IsDefault    *bool   `json:"is_default"`
}
