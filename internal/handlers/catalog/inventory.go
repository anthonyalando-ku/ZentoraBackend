package catalog

import (
	"net/http"
	"strconv"

	invdomain "zentora-service/internal/domain/inventory"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *CatalogHandler) ListLocations(c *gin.Context) {
	f := invdomain.LocationFilter{ActiveOnly: c.Query("active_only") == "true"}
	locs, err := h.svc.ListLocations(c.Request.Context(), f)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "locations retrieved", locs)
}

func (h *CatalogHandler) GetLocation(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	l, err := h.svc.GetLocationByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "location retrieved", l)
}

func (h *CatalogHandler) CreateLocation(c *gin.Context) {
	var req invdomain.CreateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	l, err := h.svc.CreateLocation(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "location created", l)
}

func (h *CatalogHandler) UpdateLocation(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req invdomain.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	l, err := h.svc.UpdateLocation(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "location updated", l)
}

func (h *CatalogHandler) DeleteLocation(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteLocation(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "location deleted", nil)
}

func (h *CatalogHandler) UpsertInventoryItem(c *gin.Context) {
	var req invdomain.UpsertItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	item, err := h.svc.UpsertInventoryItem(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "inventory item updated", item)
}

func (h *CatalogHandler) GetInventoryByVariant(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	items, err := h.svc.GetInventoryByVariant(c.Request.Context(), variantID)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "inventory retrieved", items)
}

func (h *CatalogHandler) GetStockSummary(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	summary, err := h.svc.GetStockSummary(c.Request.Context(), variantID)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "stock summary retrieved", summary)
}

func (h *CatalogHandler) AdjustAvailableStock(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	locationID, err := parseID(c, "location_id")
	if err != nil {
		return
	}
	var req invdomain.AdjustQtyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if err := h.svc.AdjustAvailableStock(c.Request.Context(), variantID, locationID, &req); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "stock adjusted", nil)
}

func (h *CatalogHandler) ReserveStock(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	locationID, err := parseID(c, "location_id")
	if err != nil {
		return
	}
	qty, err := strconv.Atoi(c.Query("qty"))
	if err != nil || qty <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid qty", nil)
		return
	}
	if err := h.svc.ReserveStock(c.Request.Context(), variantID, locationID, qty); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "stock reserved", nil)
}

func (h *CatalogHandler) ReleaseStock(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	locationID, err := parseID(c, "location_id")
	if err != nil {
		return
	}
	qty, err := strconv.Atoi(c.Query("qty"))
	if err != nil || qty <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid qty", nil)
		return
	}
	if err := h.svc.ReleaseStock(c.Request.Context(), variantID, locationID, qty); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "stock released", nil)
}

func (h *CatalogHandler) DeleteInventoryItem(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	locationID, err := parseID(c, "location_id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteInventoryItem(c.Request.Context(), variantID, locationID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "inventory item deleted", nil)
}
