// internal/service/catalog/service.go
package catalog

import (
	"zentora-service/internal/repository/postgres"
	imagekit "github.com/imagekit-developer/imagekit-go/v2"
)

// CatalogService provides business logic for the product catalog.
type CatalogService struct {
	categoryRepo  *postgres.CategoryRepository
	brandRepo     *postgres.BrandRepository
	tagRepo       *postgres.TagRepository
	productRepo   *postgres.ProductRepository
	attributeRepo *postgres.AttributeRepository
	variantRepo   *postgres.VariantRepository
	inventoryRepo *postgres.InventoryRepository
	discountRepo  *postgres.DiscountRepository
	imageKit      *imagekit.Client
}

// NewCatalogService creates a new CatalogService.
func NewCatalogService(
	categoryRepo *postgres.CategoryRepository,
	brandRepo *postgres.BrandRepository,
	tagRepo *postgres.TagRepository,
	productRepo *postgres.ProductRepository,
	attributeRepo *postgres.AttributeRepository,
	variantRepo *postgres.VariantRepository,
	inventoryRepo *postgres.InventoryRepository,
	discountRepo *postgres.DiscountRepository,
	imageKit      *imagekit.Client,
) *CatalogService {
	return &CatalogService{
		categoryRepo:  categoryRepo,
		brandRepo:     brandRepo,
		tagRepo:       tagRepo,
		productRepo:   productRepo,
		attributeRepo: attributeRepo,
		variantRepo:   variantRepo,
		inventoryRepo: inventoryRepo,
		discountRepo:  discountRepo,
		imageKit:      imageKit,
	}
}
