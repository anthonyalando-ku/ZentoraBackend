//   GET  /admin/merchant/feed/full         — generate + stream XML feed
//   GET  /admin/merchant/feed/incremental  — incremental feed (query: since=RFC3339)
//   GET  /admin/merchant/feed/stats        — count eligible variants
//   GET  /admin/merchant/feed/ping         — health check
package merchant

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"zentora-service/internal/merchant/application/feedgen"
	"zentora-service/internal/merchant/application/feedxml"
	"zentora-service/internal/merchant/domain"
	merchantsvc "zentora-service/internal/merchant/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler serves merchant feed admin endpoints.
type Handler struct {
	svc    *merchantsvc.MerchantFeedService
	logger *zap.Logger
}

// NewHandler constructs the merchant feed handler.
func NewHandler(svc *merchantsvc.MerchantFeedService, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes attaches the routes to a Gin RouterGroup.
//
//   admin := r.Group("/admin", authMiddleware.RequireAdmin())
//   merchantHandler.RegisterRoutes(admin)
func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	g := admin.Group("/feed")
	{
		g.GET("/full",        h.GenerateFull)
		g.GET("/incremental", h.GenerateIncremental)
		g.GET("/stats",       h.Stats)
		g.GET("/ping",        h.Ping)
	}
}


func (h *Handler) RegisterPublicRoutes(r *gin.Engine) {
	r.GET("/feeds/google-merchant.xml", h.PublicFeed)
}

// GenerateFull streams the complete catalog as GMC RSS 2.0 XML.
// Add ?download=1 to receive it as a file download.
func (h *Handler) GenerateFull(c *gin.Context) {
	result, err := h.svc.GenerateFull(c.Request.Context())
	if err != nil {
		h.logger.Error("merchant feed full generation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feed generation failed"})
		return
	}
	h.streamXML(c, result, "zentora-merchant-feed-full.xml")
}

// GenerateIncremental streams only products updated after ?since=RFC3339.
// Defaults to the last 24 hours when since is omitted.
func (h *Handler) GenerateIncremental(c *gin.Context) {
	sinceStr := c.Query("since")
	var since time.Time

	if sinceStr == "" {
		since = time.Now().UTC().Add(-24 * time.Hour)
	} else {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid 'since' param — use RFC3339, got %q", sinceStr),
			})
			return
		}
	}

	result, err := h.svc.GenerateIncremental(c.Request.Context(), since)
	if err != nil {
		h.logger.Error("merchant feed incremental generation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feed generation failed"})
		return
	}

	filename := fmt.Sprintf("zentora-merchant-feed-incremental-%s.xml",
		time.Now().UTC().Format("20060102-1504"))
	h.streamXML(c, result, filename)
}

func (h *Handler) PublicFeed(c *gin.Context) {
	result, err := h.svc.GenerateFull(c.Request.Context())
	if err != nil {
		h.logger.Error("merchant public feed generation failed", zap.Error(err))
		// Return 503 so Google retries rather than treating it as an empty feed
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feed temporarily unavailable"})
		return
	}
 
	// Build a weak ETag from the generation timestamp + item count so Google's
	// crawler can skip downloading an unchanged feed.
	etag := fmt.Sprintf(`W/"%s-%d"`,
		result.Metadata.GeneratedAt.UTC().Format("20060102150405"),
		result.Metadata.ValidItems,
	)
 
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
 
	// Cache for 1 hour at the CDN / reverse proxy level.
	// Google re-fetches on its own schedule regardless; this just avoids
	// hammering the DB if multiple crawlers or health-checks hit the URL.
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("ETag",          etag)
 
	h.streamXML(c, result, "google-merchant.xml")
}

// Stats returns feed eligibility counts as JSON (no feed generation).
func (h *Handler) Stats(c *gin.Context) {
	filter := merchant.DefaultFeedFilter()
	count, err := h.svc.Repo().CountEligibleVariants(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("merchant feed stats failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stats unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"eligible_variants": count,
		"batch_size":        filter.Limit,
		"estimated_batches": (count + int64(filter.Limit) - 1) / int64(filter.Limit),
		"checked_at":        time.Now().UTC().Format(time.RFC3339),
	})
}

// Ping is a lightweight connectivity check.
func (h *Handler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"service": "merchant-feed", "status": "ok"})
}

// streamXML is the shared response writer for both full and incremental feeds.
func (h *Handler) streamXML(c *gin.Context, result *feedgen.GenerateResult, filename string) {
	var allItems []merchant.MerchantFeedItem
	for _, batch := range result.Batches {
		allItems = append(allItems, batch.Items...)
	}

	feed := feedxml.BuildRSSFeed(
		result.Metadata,
		allItems,
		h.svc.StoreBaseURL(),
		h.svc.StoreTitle(),
	)

	if c.Query("download") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	}
	c.Header("Content-Type",          "application/rss+xml; charset=utf-8")
	c.Header("X-Feed-Total-Items",    fmt.Sprintf("%d", result.Metadata.TotalItems))
	c.Header("X-Feed-Valid-Items",    fmt.Sprintf("%d", result.Metadata.ValidItems))
	c.Header("X-Feed-Invalid-Items",  fmt.Sprintf("%d", result.Metadata.InvalidItems))
	c.Header("X-Feed-Generated-At",   result.Metadata.GeneratedAt.Format(time.RFC3339))
	c.Header("X-Feed-Duration-Ms",    fmt.Sprintf("%.0f", result.Metadata.DurationSeconds*1000))
	c.Status(http.StatusOK)

	_, _ = c.Writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"))
	enc := xml.NewEncoder(c.Writer)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		h.logger.Error("merchant feed XML encode error", zap.Error(err))
	}
}