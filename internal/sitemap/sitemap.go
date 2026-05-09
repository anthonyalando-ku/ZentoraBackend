package sitemap

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	siteURL      = "https://zentorashop.co.ke"
	apiURL       = "https://zentora-api.onrender.com"
	urlsPerChunk = 1_000
)

// Handler serves all sitemap endpoints.
type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

// RegisterRoutes wires sitemap endpoints onto the root engine (no auth).
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/sitemap-index.xml", h.SitemapIndex)
	r.GET("/sitemaps/products-:chunk.xml", h.ProductSitemap)
}

// ---------------------------------------------------------------------------
// XML structs
// ---------------------------------------------------------------------------

type URLSet struct {
	XMLName xml.Name   `xml:"urlset"`
	XMLNS   string     `xml:"xmlns,attr"`
	XSI     string     `xml:"xmlns:xsi,attr,omitempty"`
	Schema  string     `xml:"xsi:schemaLocation,attr,omitempty"`
	URLs    []SitemapURL `xml:"url"`
}

type SitemapURL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

type SitemapIndex struct {
	XMLName  xml.Name        `xml:"sitemapindex"`
	XMLNS    string          `xml:"xmlns,attr"`
	Sitemaps []SitemapEntry  `xml:"sitemap"`
}

type SitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}


func (h *Handler) SitemapIndex(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	total, err := h.countActiveProducts(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Printf("Error counting products for sitemap: %v", err)
		return
	}

	today := time.Now().UTC().Format("2006-01-02")
	chunks := int(math.Ceil(float64(total) / float64(urlsPerChunk)))
	if chunks == 0 {
		chunks = 1
	}

	idx := SitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	// Static pages sitemap (served from /public in the frontend)
	idx.Sitemaps = append(idx.Sitemaps, SitemapEntry{
		Loc:     siteURL + "/sitemap-pages.xml",
		LastMod: today,
	})

	// Category sitemap
	idx.Sitemaps = append(idx.Sitemaps, SitemapEntry{
		Loc:     siteURL + "/sitemap-categories.xml",
		LastMod: today,
	})

	// Product chunk sitemaps (served from the API)
	for i := 1; i <= chunks; i++ {
		idx.Sitemaps = append(idx.Sitemaps, SitemapEntry{
			Loc:     fmt.Sprintf("%s/sitemaps/products-%d.xml", apiURL, i),
			LastMod: today,
		})
	}

	writeXML(c, idx)
}

// ---------------------------------------------------------------------------
// GET /sitemaps/products-:chunk.xml
// ---------------------------------------------------------------------------
func (h *Handler) ProductSitemap(c *gin.Context) {
	rawChunk := c.Param("chunk.xml")

	rawChunk = strings.TrimSuffix(rawChunk, ".xml")

	log.Printf("Generating sitemap chunk %s", rawChunk)

	chunk, err := strconv.Atoi(rawChunk)
	if err != nil || chunk < 1 {
		c.Status(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	offset := (chunk - 1) * urlsPerChunk

	type row struct {
		Slug      string
		UpdatedAt time.Time
	}

	const q = `
		SELECT p.slug, p.updated_at
		FROM products p
		WHERE p.status = 'active'
		ORDER BY p.updated_at DESC, p.id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := h.db.Query(ctx, q, urlsPerChunk, offset)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	urlset := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Slug, &r.UpdatedAt); err != nil {
			continue
		}
		urlset.URLs = append(urlset.URLs, SitemapURL{
			Loc:        fmt.Sprintf("%s/products/%s", siteURL, r.Slug),
			LastMod:    r.UpdatedAt.UTC().Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   0.8,
		})
	}

	if err := rows.Err(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Cache aggressively — Google doesn't need real-time sitemaps.
	// Regenerates naturally on next crawl after 4 hours.
	c.Header("Cache-Control", "public, max-age=14400")
	writeXML(c, urlset)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (h *Handler) countActiveProducts(ctx context.Context) (int, error) {
	var count int
	err := h.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM products WHERE status = 'active'`,
	).Scan(&count)
	return count, err
}

func writeXML(c *gin.Context, v any) {
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(http.StatusOK)

	_, _ = c.Writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"))

	enc := xml.NewEncoder(c.Writer)
	enc.Indent("", "  ")
	_ = enc.Encode(v)
}