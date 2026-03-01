// internal/domain/catalog/dto.go
package catalog

// ===========================
// Category DTOs
// ===========================

type CreateCategoryRequest struct {
	Name     string `json:"name" binding:"required,max=255"`
	Slug     string `json:"slug" binding:"required,max=255"`
	ParentID *int64 `json:"parent_id"`
	IsActive *bool  `json:"is_active"`
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=255"`
	Slug     *string `json:"slug" binding:"omitempty,max=255"`
	ParentID *int64  `json:"parent_id"`
	IsActive *bool   `json:"is_active"`
}

// ===========================
// Brand DTOs
// ===========================

type CreateBrandRequest struct {
	Name    string  `json:"name" binding:"required,max=255"`
	Slug    string  `json:"slug" binding:"required,max=255"`
	LogoURL *string `json:"logo_url"`
}

type UpdateBrandRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=255"`
	Slug     *string `json:"slug" binding:"omitempty,max=255"`
	LogoURL  *string `json:"logo_url"`
	IsActive *bool   `json:"is_active"`
}

// ===========================
// Product DTOs
// ===========================

type CreateProductRequest struct {
	Name             string   `json:"name" binding:"required,max=255"`
	Slug             string   `json:"slug" binding:"required,max=255"`
	Description      *string  `json:"description"`
	ShortDescription *string  `json:"short_description"`
	BrandID          *int64   `json:"brand_id"`
	BasePrice        float64  `json:"base_price" binding:"required,gt=0"`
	Status           string   `json:"status" binding:"omitempty,oneof=active draft archived"`
	IsFeatured       bool     `json:"is_featured"`
	IsDigital        bool     `json:"is_digital"`
	CategoryIDs      []int64  `json:"category_ids"`
	TagNames         []string `json:"tag_names"`
}

type UpdateProductRequest struct {
	Name             *string  `json:"name" binding:"omitempty,max=255"`
	Slug             *string  `json:"slug" binding:"omitempty,max=255"`
	Description      *string  `json:"description"`
	ShortDescription *string  `json:"short_description"`
	BrandID          *int64   `json:"brand_id"`
	BasePrice        *float64 `json:"base_price" binding:"omitempty,gt=0"`
	Status           *string  `json:"status" binding:"omitempty,oneof=active draft archived"`
	IsFeatured       *bool    `json:"is_featured"`
	IsDigital        *bool    `json:"is_digital"`
}

type SetProductTagsRequest struct {
	TagNames []string `json:"tag_names" binding:"required"`
}

type SetProductCategoriesRequest struct {
	CategoryIDs []int64 `json:"category_ids" binding:"required"`
}

type SetProductAttributeValuesRequest struct {
	AttributeValueIDs []int64 `json:"attribute_value_ids" binding:"required"`
}

// ===========================
// Product Image DTOs
// ===========================

type AddProductImageRequest struct {
	ImageURL  string `json:"image_url" binding:"required,max=500"`
	IsPrimary bool   `json:"is_primary"`
	SortOrder int    `json:"sort_order"`
}

// ===========================
// Attribute DTOs
// ===========================

type CreateAttributeRequest struct {
	Name               string `json:"name" binding:"required,max=100"`
	Slug               string `json:"slug" binding:"required,max=100"`
	IsVariantDimension bool   `json:"is_variant_dimension"`
}

type UpdateAttributeRequest struct {
	Name               *string `json:"name" binding:"omitempty,max=100"`
	Slug               *string `json:"slug" binding:"omitempty,max=100"`
	IsVariantDimension *bool   `json:"is_variant_dimension"`
}

type CreateAttributeValueRequest struct {
	Value string `json:"value" binding:"required,max=100"`
}

// ===========================
// Variant DTOs
// ===========================

type CreateVariantRequest struct {
	SKU               string   `json:"sku" binding:"required,max=100"`
	Price             float64  `json:"price" binding:"required,gt=0"`
	Weight            *float64 `json:"weight"`
	IsActive          bool     `json:"is_active"`
	AttributeValueIDs []int64  `json:"attribute_value_ids"`
}

type UpdateVariantRequest struct {
	SKU      *string  `json:"sku" binding:"omitempty,max=100"`
	Price    *float64 `json:"price" binding:"omitempty,gt=0"`
	Weight   *float64 `json:"weight"`
	IsActive *bool    `json:"is_active"`
}

type SetVariantAttributeValuesRequest struct {
	AttributeValueIDs []int64 `json:"attribute_value_ids" binding:"required"`
}
