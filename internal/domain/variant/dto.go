package variant

import (
	"strings"
	"unicode/utf8"
)

type CreateRequest struct {
	SKU               string   `json:"sku"`
	Price             float64  `json:"price"`
	Weight            *float64 `json:"weight,omitempty"`
	IsActive          *bool    `json:"is_active,omitempty"`
	AttributeValueIDs []int64  `json:"attribute_value_ids,omitempty"`
}

func (r *CreateRequest) Validate() error {
	r.SKU = strings.TrimSpace(r.SKU)
	if r.SKU == "" || utf8.RuneCountInString(r.SKU) > 100 {
		return ErrInvalidSKU
	}
	if r.Price <= 0 {
		return ErrInvalidPrice
	}
	if r.Weight != nil && *r.Weight <= 0 {
		return ErrInvalidWeight
	}
	return nil
}

type UpdateRequest struct {
	SKU      *string  `json:"sku,omitempty"`
	Price    *float64 `json:"price,omitempty"`
	Weight   *float64 `json:"weight,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

func (r *UpdateRequest) Validate() error {
	if r.SKU != nil {
		*r.SKU = strings.TrimSpace(*r.SKU)
		if *r.SKU == "" || utf8.RuneCountInString(*r.SKU) > 100 {
			return ErrInvalidSKU
		}
	}
	if r.Price != nil && *r.Price <= 0 {
		return ErrInvalidPrice
	}
	if r.Weight != nil && *r.Weight <= 0 {
		return ErrInvalidWeight
	}
	return nil
}

type SetAttributeValuesRequest struct {
	AttributeValueIDs []int64 `json:"attribute_value_ids"`
}