// internal/domain/catalog/repository.go
package catalog

import "context"

// CategoryRepository defines persistence operations for product categories
// and the category_closure table.
type CategoryRepository interface {
	CreateCategory(ctx context.Context, c *Category) error
	GetCategoryByID(ctx context.Context, id int64) (*Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	UpdateCategory(ctx context.Context, c *Category) error
	DeleteCategory(ctx context.Context, id int64) error
	ListCategories(ctx context.Context) ([]Category, error)
	GetCategoryAncestors(ctx context.Context, id int64) ([]CategoryClosure, error)
	GetCategoryDescendants(ctx context.Context, id int64) ([]CategoryClosure, error)
}

// BrandRepository defines persistence operations for product brands.
type BrandRepository interface {
	CreateBrand(ctx context.Context, b *Brand) error
	GetBrandByID(ctx context.Context, id int64) (*Brand, error)
	GetBrandBySlug(ctx context.Context, slug string) (*Brand, error)
	UpdateBrand(ctx context.Context, b *Brand) error
	DeleteBrand(ctx context.Context, id int64) error
	ListBrands(ctx context.Context, activeOnly bool) ([]Brand, error)
}

// TagRepository defines persistence operations for tags.
type TagRepository interface {
	// FindOrCreateByName returns an existing tag or creates a new one within the
	// provided transaction/context.  The caller is responsible for transaction
	// management; the method operates on whichever pgx conn/tx is passed via ctx
	// conventions.  Because pgx does not propagate a tx via context, this method
	// accepts the raw pool and delegates transaction boundary to the service
	// layer.  See ProductRepository.CreateProductWithTags for transactional use.
	FindOrCreateByName(ctx context.Context, name string) (*Tag, error)
	GetTagByID(ctx context.Context, id int64) (*Tag, error)
	ListTags(ctx context.Context) ([]Tag, error)
}

// ProductRepository defines persistence operations for products, images, and
// the product-level join tables (category_map, tags, attribute_values).
type ProductRepository interface {
	CreateProduct(ctx context.Context, p *Product) error
	GetProductByID(ctx context.Context, id int64) (*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, p *Product) error
	DeleteProduct(ctx context.Context, id int64) error
	ListProducts(ctx context.Context, limit, offset int) ([]Product, int64, error)

	// Category mapping
	AddProductCategory(ctx context.Context, productID, categoryID int64) error
	RemoveProductCategory(ctx context.Context, productID, categoryID int64) error
	GetProductCategories(ctx context.Context, productID int64) ([]Category, error)

	// Tag linking – transactional: if tag name is not found it is created first.
	SetProductTags(ctx context.Context, productID int64, tagNames []string) error
	GetProductTags(ctx context.Context, productID int64) ([]Tag, error)

	// Images
	AddProductImage(ctx context.Context, img *ProductImage) error
	GetProductImages(ctx context.Context, productID int64) ([]ProductImage, error)
	DeleteProductImage(ctx context.Context, id int64) error
	SetPrimaryImage(ctx context.Context, productID, imageID int64) error

	// Attribute values (product-level, not variant)
	SetProductAttributeValues(ctx context.Context, productID int64, attributeValueIDs []int64) error
	GetProductAttributeValues(ctx context.Context, productID int64) ([]AttributeValue, error)
}

// AttributeRepository defines persistence operations for attributes and their
// values.
type AttributeRepository interface {
	CreateAttribute(ctx context.Context, a *Attribute) error
	GetAttributeByID(ctx context.Context, id int64) (*Attribute, error)
	GetAttributeBySlug(ctx context.Context, slug string) (*Attribute, error)
	UpdateAttribute(ctx context.Context, a *Attribute) error
	DeleteAttribute(ctx context.Context, id int64) error
	ListAttributes(ctx context.Context) ([]Attribute, error)

	CreateAttributeValue(ctx context.Context, v *AttributeValue) error
	GetAttributeValueByID(ctx context.Context, id int64) (*AttributeValue, error)
	ListAttributeValues(ctx context.Context, attributeID int64) ([]AttributeValue, error)
	DeleteAttributeValue(ctx context.Context, id int64) error
}

// VariantRepository defines persistence operations for product variants and
// their attribute value links.
type VariantRepository interface {
	CreateVariant(ctx context.Context, v *ProductVariant) error
	GetVariantByID(ctx context.Context, id int64) (*ProductVariant, error)
	GetVariantBySKU(ctx context.Context, sku string) (*ProductVariant, error)
	UpdateVariant(ctx context.Context, v *ProductVariant) error
	DeleteVariant(ctx context.Context, id int64) error
	ListVariantsByProduct(ctx context.Context, productID int64) ([]ProductVariant, error)

	SetVariantAttributeValues(ctx context.Context, variantID int64, attributeValueIDs []int64) error
	GetVariantAttributeValues(ctx context.Context, variantID int64) ([]AttributeValue, error)
}
