package attribute

import (
	"strings"
	"unicode/utf8"
)

type CreateRequest struct {
	Name               string `json:"name"`
	IsVariantDimension bool   `json:"is_variant_dimension"`
}

func (r *CreateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 100 {
		return ErrInvalidName
	}
	return nil
}

type UpdateRequest struct {
	Name               *string `json:"name,omitempty"`
	IsVariantDimension *bool   `json:"is_variant_dimension,omitempty"`
}

func (r *UpdateRequest) Validate() error {
	if r.Name != nil {
		*r.Name = strings.TrimSpace(*r.Name)
		if *r.Name == "" || utf8.RuneCountInString(*r.Name) > 100 {
			return ErrInvalidName
		}
	}
	return nil
}

type CreateValueRequest struct {
	Value string `json:"value"`
}

func (r *CreateValueRequest) Validate() error {
	r.Value = strings.TrimSpace(r.Value)
	if r.Value == "" || utf8.RuneCountInString(r.Value) > 100 {
		return ErrInvalidValue
	}
	return nil
}

type SetProductAttributeValuesRequest struct {
	AttributeValueIDs []int64 `json:"attribute_value_ids"`
}