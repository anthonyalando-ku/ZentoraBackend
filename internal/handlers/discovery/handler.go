package discovery

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	categorydomain "zentora-service/internal/domain/category"
	discoverydomain "zentora-service/internal/domain/discovery"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type discoveryService interface {
	GetFeed(ctx context.Context, req *discoverydomain.FeedRequest) ([]discoverydomain.ProductCard, error)
	Suggest(ctx context.Context, req *discoverydomain.SuggestRequest) ([]discoverydomain.Suggestion, error)
	TrackSearch(ctx context.Context, event *discoverydomain.SearchEvent) (int64, error)
	TrackSearchClick(ctx context.Context, event *discoverydomain.SearchClickEvent) error
}

type metricsRunner interface {
	RunOnce(ctx context.Context) error
}

type Handler struct {
	discovery discoveryService
	metrics   metricsRunner
}

func NewHandler(discovery discoveryService, metrics metricsRunner) *Handler {
	return &Handler{
		discovery: discovery,
		metrics:   metrics,
	}
}

func (h *Handler) GetFeedCandidates(c *gin.Context) {
	req, err := buildFeedRequest(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	items, err := h.discovery.GetFeed(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "feed retrieved", gin.H{
		"feed_type": req.FeedType,
		"limit":     req.Limit,
		"items":     items,
	})
}

func (h *Handler) Suggest(c *gin.Context) {
	limit, err := parseOptionalInt(c.Query("limit"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	suggestions, err := h.discovery.Suggest(c.Request.Context(), &discoverydomain.SuggestRequest{
		Prefix: c.Query("prefix"),
		Limit:  limit,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "suggestions retrieved", gin.H{
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}

type trackSearchRequest struct {
	Query       string                      `json:"query"`
	SessionID   *string                     `json:"session_id"`
	ResultCount int                         `json:"result_count"`
	Results     []searchResultPositionInput `json:"results"`
}

type searchResultPositionInput struct {
	ProductID int64   `json:"product_id"`
	Position  int     `json:"position"`
	Score     float64 `json:"score"`
}

func (h *Handler) TrackSearch(c *gin.Context) {
	var req trackSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	event := &discoverydomain.SearchEvent{
		Query:       req.Query,
		SessionID:   req.SessionID,
		ResultCount: req.ResultCount,
	}
	if len(req.Results) > 0 {
		event.Results = make([]discoverydomain.SearchResultPosition, 0, len(req.Results))
		for _, result := range req.Results {
			event.Results = append(event.Results, discoverydomain.SearchResultPosition{
				ProductID: result.ProductID,
				Position:  result.Position,
				Score:     result.Score,
			})
		}
	}
	if identityID, ok := middleware.GetIdentityID(c); ok {
		event.UserID = &identityID
	}

	eventID, err := h.discovery.TrackSearch(c.Request.Context(), event)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "search tracked", gin.H{
		"search_event_id": eventID,
	})
}

type trackSearchClickRequest struct {
	SearchEventID int64   `json:"search_event_id"`
	ProductID     int64   `json:"product_id"`
	Position      int     `json:"position"`
	SessionID     *string `json:"session_id"`
}

func (h *Handler) TrackSearchClick(c *gin.Context) {
	var req trackSearchClickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	event := &discoverydomain.SearchClickEvent{
		SearchEventID: req.SearchEventID,
		ProductID:     req.ProductID,
		Position:      req.Position,
		SessionID:     req.SessionID,
	}
	if identityID, ok := middleware.GetIdentityID(c); ok {
		event.UserID = &identityID
	}

	if err := h.discovery.TrackSearchClick(c.Request.Context(), event); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "search click tracked", nil)
}

func (h *Handler) RecomputeMetrics(c *gin.Context) {
	if err := h.metrics.RunOnce(c.Request.Context()); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to recompute discovery metrics", err)
		return
	}

	response.Success(c, http.StatusOK, "discovery metrics recomputed", nil)
}

func buildFeedRequest(c *gin.Context) (*discoverydomain.FeedRequest, error) {
	limit, err := parseOptionalInt(c.Query("limit"))
	if err != nil {
		return nil, err
	}

	req := &discoverydomain.FeedRequest{
		FeedType: discoverydomain.FeedType(strings.TrimSpace(c.Query("feed_type"))),
		Limit:    limit,
	}

	if identityID, ok := middleware.GetIdentityID(c); ok {
		req.UserID = &identityID
	}

	if sessionID := strings.TrimSpace(c.Query("session_id")); sessionID != "" {
		req.SessionID = &sessionID
	}
	if query := strings.TrimSpace(c.Query("query")); query != "" {
		req.Query = &query
	}

	if categoryID := strings.TrimSpace(c.Query("category_id")); categoryID != "" {
		id, err := strconv.ParseInt(categoryID, 10, 64)
		if err != nil {
			return nil, errors.New("category_id must be a valid integer")
		}
		req.CategoryID = &id
	}

	brandIDs, err := parseCSVInt64(c.Query("brand_ids"))
	if err != nil {
		return nil, err
	}
	tagIDs, err := parseCSVInt64(c.Query("tag_ids"))
	if err != nil {
		return nil, err
	}
	variantAttributeValueIDs, err := parseCSVInt64(c.Query("variant_attribute_value_ids"))
	if err != nil {
		return nil, err
	}
	priceMin, err := parseOptionalFloat64(c.Query("price_min"))
	if err != nil {
		return nil, err
	}
	priceMax, err := parseOptionalFloat64(c.Query("price_max"))
	if err != nil {
		return nil, err
	}
	minRating, err := parseOptionalFloat64(c.Query("min_rating"))
	if err != nil {
		return nil, err
	}

	req.Filters = discoverydomain.FeedFilter{
		BrandIDs:                 brandIDs,
		TagIDs:                   tagIDs,
		VariantAttributeValueIDs: variantAttributeValueIDs,
		PriceMin:                 priceMin,
		PriceMax:                 priceMax,
		MinRating:                minRating,
		DiscountOnly:             c.Query("discount_only") == "true",
		InStockOnly:              c.Query("in_stock_only") == "true",
	}

	return req, nil
}

func parseOptionalInt(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("value must be a valid integer")
	}
	return value, nil
}

func parseCSVInt64(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	values := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, errors.New("list values must be valid integers")
		}
		values = append(values, value)
	}
	return values, nil
}

func parseOptionalFloat64(raw string) (*float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, errors.New("value must be a valid number")
	}
	return &value, nil
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, discoverydomain.ErrInvalidRequest),
		errors.Is(err, discoverydomain.ErrInvalidFeedType),
		errors.Is(err, discoverydomain.ErrCategoryRequired),
		errors.Is(err, discoverydomain.ErrInvalidCategoryID),
		errors.Is(err, discoverydomain.ErrInvalidUserID),
		errors.Is(err, discoverydomain.ErrUserRequired),
		errors.Is(err, discoverydomain.ErrUserOrSessionRequired),
		errors.Is(err, discoverydomain.ErrInvalidSearchEvent),
		errors.Is(err, discoverydomain.ErrInvalidProductID),
		errors.Is(err, discoverydomain.ErrInvalidPosition),
		errors.Is(err, discoverydomain.ErrNegativeResultCount),
		errors.Is(err, discoverydomain.ErrQueryRequired),
		errors.Is(err, discoverydomain.ErrPrefixRequired):
		response.Error(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, categorydomain.ErrNotFound):
		response.Error(c, http.StatusNotFound, "not found", err)
	default:
		response.Error(c, http.StatusInternalServerError, "internal server error", err)
	}
}
