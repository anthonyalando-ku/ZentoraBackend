package brand

import (
	"strings"
	"unicode/utf8"
)

type CreateRequest struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

func (r *CreateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 255 {
		return ErrInvalidName
	}
	if r.LogoURL != nil && utf8.RuneCountInString(*r.LogoURL) > 500 {
		return ErrInvalidLogo
	}
	return nil
}

type UpdateRequest struct {
	Name     *string `json:"name,omitempty"`
	LogoURL  *string `json:"logo_url,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

func (r *UpdateRequest) Validate() error {
	if r.Name != nil {
		*r.Name = strings.TrimSpace(*r.Name)
		if *r.Name == "" || utf8.RuneCountInString(*r.Name) > 255 {
			return ErrInvalidName
		}
	}
	if r.LogoURL != nil && utf8.RuneCountInString(*r.LogoURL) > 500 {
		return ErrInvalidLogo
	}
	return nil
}

type ListFilter struct {
	ActiveOnly bool
}