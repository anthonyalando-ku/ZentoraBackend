// internal/handlers/catalog/handler.go
package catalog

import (
	"errors"
	"net/http"
	"strconv"

	attrdomain "zentora-service/internal/domain/attribute"
	branddomain "zentora-service/internal/domain/brand"
	categorydomain "zentora-service/internal/domain/category"
	discdomain "zentora-service/internal/domain/discount"
	invdomain "zentora-service/internal/domain/inventory"
	productdomain "zentora-service/internal/domain/product"
	tagdomain "zentora-service/internal/domain/tag"
	variantdomain "zentora-service/internal/domain/variant"
	xerrors "zentora-service/internal/pkg/errors"
	"zentora-service/internal/pkg/response"
	catalogSvc "zentora-service/internal/service/catalog"

	"github.com/gin-gonic/gin"
)

// CatalogHandler holds HTTP handlers for the catalog domain.
type CatalogHandler struct {
	svc *catalogSvc.CatalogService
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(svc *catalogSvc.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func parseID(c *gin.Context, param string) (int64, error) {
	raw := c.Param(param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid "+param, err)
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, categorydomain.ErrNotFound),
		errors.Is(err, branddomain.ErrNotFound),
		errors.Is(err, tagdomain.ErrNotFound),
		errors.Is(err, attrdomain.ErrNotFound),
		errors.Is(err, attrdomain.ErrValueNotFound),
		errors.Is(err, variantdomain.ErrNotFound),
		errors.Is(err, invdomain.ErrLocationNotFound),
		errors.Is(err, invdomain.ErrItemNotFound),
		errors.Is(err, discdomain.ErrNotFound),
		errors.Is(err, discdomain.ErrRedemptionNotFound),
		errors.Is(err, productdomain.ErrNotFound),
		errors.Is(err, productdomain.ErrImageNotFound),
		errors.Is(err, xerrors.ErrNotFound):
		response.Error(c, http.StatusNotFound, "not found", nil)

	case errors.Is(err, categorydomain.ErrSlugConflict),
		errors.Is(err, branddomain.ErrSlugConflict),
		errors.Is(err, branddomain.ErrNameConflict),
		errors.Is(err, attrdomain.ErrSlugConflict),
		errors.Is(err, attrdomain.ErrValueConflict),
		errors.Is(err, variantdomain.ErrSKUConflict),
		errors.Is(err, invdomain.ErrLocationCodeConflict),
		errors.Is(err, invdomain.ErrItemConflict),
		errors.Is(err, discdomain.ErrCodeConflict),
		errors.Is(err, discdomain.ErrRedemptionConflict),
		errors.Is(err, productdomain.ErrSlugConflict):
		response.Error(c, http.StatusConflict, err.Error(), nil)

	case errors.Is(err, invdomain.ErrInsufficientStock),
		errors.Is(err, categorydomain.ErrCircularParent),
		errors.Is(err, discdomain.ErrExpired),
		errors.Is(err, discdomain.ErrNotStarted),
		errors.Is(err, discdomain.ErrInactive),
		errors.Is(err, discdomain.ErrMaxRedemptions),
		errors.Is(err, discdomain.ErrMinOrderAmount):
		response.Error(c, http.StatusUnprocessableEntity, err.Error(), nil)

	case errors.Is(err, categorydomain.ErrInvalidName),
		errors.Is(err, categorydomain.ErrInvalidParent),
		errors.Is(err, branddomain.ErrInvalidName),
		errors.Is(err, branddomain.ErrInvalidLogo),
		errors.Is(err, tagdomain.ErrInvalidName),
		errors.Is(err, attrdomain.ErrInvalidName),
		errors.Is(err, attrdomain.ErrInvalidValue),
		errors.Is(err, attrdomain.ErrInvalidSlug),
		errors.Is(err, variantdomain.ErrInvalidSKU),
		errors.Is(err, variantdomain.ErrInvalidPrice),
		errors.Is(err, variantdomain.ErrInvalidWeight),
		errors.Is(err, invdomain.ErrInvalidName),
		errors.Is(err, invdomain.ErrInvalidLocationCode),
		errors.Is(err, invdomain.ErrInvalidQuantity),
		errors.Is(err, discdomain.ErrInvalidName),
		errors.Is(err, discdomain.ErrInvalidCode),
		errors.Is(err, discdomain.ErrInvalidType),
		errors.Is(err, discdomain.ErrInvalidValue),
		errors.Is(err, discdomain.ErrInvalidPercentage),
		errors.Is(err, discdomain.ErrInvalidDateRange),
		errors.Is(err, discdomain.ErrInvalidTargetType),
		errors.Is(err, productdomain.ErrInvalidName),
		errors.Is(err, productdomain.ErrInvalidPrice),
		errors.Is(err, productdomain.ErrInvalidStatus),
		errors.Is(err, productdomain.ErrBrandRequired),
		errors.Is(err, productdomain.ErrCategoryRequired),
		errors.Is(err, productdomain.ErrImageRequired),
		errors.Is(err, productdomain.ErrVariantRequired):
		response.Error(c, http.StatusBadRequest, err.Error(), nil)

	default:
		response.Error(c, http.StatusInternalServerError, "internal server error", nil)
	}
}
