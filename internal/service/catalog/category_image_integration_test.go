package catalog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	imagekit "github.com/imagekit-developer/imagekit-go/v2"
	"github.com/imagekit-developer/imagekit-go/v2/option"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"zentora-service/internal/domain/category"
	handler "zentora-service/internal/handlers/catalog"
	"zentora-service/internal/repository/postgres"
	catalog "zentora-service/internal/service/catalog"
)

type imageKitTransport struct {
	t                *testing.T
	uploads, deletes int
	fail             bool
}

func (m *imageKitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	status, body := 200, `{"url":"https://ik.imagekit.io/test/category.webp","fileId":"new-file"}`
	if req.Method == http.MethodDelete {
		m.deletes++
		status = 204
		body = ""
	} else {
		m.uploads++
		if err := req.ParseMultipartForm(10 << 20); err != nil {
			m.t.Fatal(err)
		}
		defer req.MultipartForm.RemoveAll()
		f, fh, err := req.FormFile("file")
		if err != nil {
			m.t.Fatal(err)
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			m.t.Fatal(err)
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || format != "webp" || cfg.Width != 800 || cfg.Height != 400 || fh.Size == 0 {
			m.t.Fatalf("upload was not optimized WebP: %+v %s %v", cfg, format, err)
		}
		if req.FormValue("folder") != "/zentora" {
			m.t.Fatal("did not reuse upload folder")
		}
		if m.fail {
			status = 500
			body = `{"message":"mock upload failure"}`
		}
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func TestCategoryImagesEndToEnd(t *testing.T) {
	url := os.Getenv("CATEGORY_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set CATEGORY_TEST_DATABASE_URL to a disposable localhost PostgreSQL cluster")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Host != "127.0.0.1" && cfg.ConnConfig.Host != "localhost" {
		t.Fatal("local database only")
	}
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("category_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	source, err := os.ReadFile("../../db/migrations/001_svc_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(source)
	start := strings.Index(sql, "CREATE TABLE product_categories (")
	end := strings.Index(sql, "-- BRANDS")
	if _, err = db.Exec(ctx, sql[start:end]); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO product_categories (name,slug) VALUES ('Legacy','legacy'); CREATE TABLE product_category_map (product_id BIGINT, category_id BIGINT);`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../db/migrations/010_category_images.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	repo := postgres.NewCategoryRepository(db)
	old, err := repo.GetCategoryBySlug(ctx, "legacy")
	if err != nil || old.ImageURL != nil {
		t.Fatalf("legacy migration: %+v %v", old, err)
	}
	transport := &imageKitTransport{t: t}
	ik := imagekit.NewClient(option.WithPrivateKey("test-only"), option.WithHTTPClient(&http.Client{Transport: transport}), option.WithMaxRetries(0))
	svc := catalog.NewCatalogService(repo, nil, nil, nil, nil, nil, nil, nil, &ik)
	h := handler.NewCatalogHandler(svc, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/categories", h.CreateCategory)
	router.PUT("/categories/:id", h.UpdateCategory)
	router.GET("/categories", h.ListCategories)
	router.GET("/categories/:id", h.GetCategory)
	var raw bytes.Buffer
	if err := png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 1200, 600))); err != nil {
		t.Fatal(err)
	}
	request := func(method, path, data string, upload []byte, want int) *category.Category {
		t.Helper()
		var req *http.Request
		if upload == nil {
			req = httptest.NewRequest(method, path, strings.NewReader(data))
			req.Header.Set("Content-Type", "application/json")
		} else {
			var body bytes.Buffer
			w := multipart.NewWriter(&body)
			_ = w.WriteField("data", data)
			f, _ := w.CreateFormFile("image", "category.png")
			_, _ = f.Write(upload)
			_ = w.Close()
			req = httptest.NewRequest(method, path, &body)
			req.Header.Set("Content-Type", w.FormDataContentType())
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		var result struct{ Data category.Category }
		if want < 300 {
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
		}
		return &result.Data
	}
	plain := request("POST", "/categories", `{"name":"No image"}`, nil, 201)
	if plain.ImageURL != nil {
		t.Fatal("image required/defaulted")
	}
	created := request("POST", "/categories", `{"name":"Uploaded"}`, raw.Bytes(), 201)
	if created.ImageURL == nil {
		t.Fatal("uploaded image missing")
	}
	path := fmt.Sprintf("/categories/%d", old.ID)
	updated := request("PUT", path, `{}`, raw.Bytes(), 200)
	if updated.ImageURL == nil {
		t.Fatal("existing category upload missing")
	}
	request("PUT", path, `{"image_url":"https://example.com/changed.png"}`, nil, 200)
	preserved := request("PUT", path, `{"name":"Renamed"}`, nil, 200)
	if preserved.ImageURL == nil || *preserved.ImageURL != "https://example.com/changed.png" {
		t.Fatal("metadata edit lost image")
	}
	request("PUT", path, `{"image_url":"javascript:alert(1)"}`, nil, 400)
	request("PUT", path, `{}`, []byte("not an image"), 400)
	transport.fail = true
	request("PUT", path, `{"name":"Must not save"}`, raw.Bytes(), 502)
	transport.fail = false
	kept, _ := repo.GetCategoryByID(ctx, old.ID)
	if kept.Name != "Renamed" || *kept.ImageURL != "https://example.com/changed.png" {
		t.Fatal("failed operation corrupted category")
	}
	removed := request("PUT", path, `{"image_url":""}`, nil, 200)
	if removed.ImageURL != nil {
		t.Fatal("remove failed")
	}
	// A DB failure after an upload must clean up only the newly uploaded file.
	_, err = db.Exec(ctx, `ALTER TABLE product_categories ADD CONSTRAINT reject_test_name CHECK (name <> 'Reject')`)
	if err != nil {
		t.Fatal(err)
	}
	request("POST", "/categories", `{"name":"Reject"}`, raw.Bytes(), 500)
	if transport.deletes != 1 {
		t.Fatalf("orphan cleanup count %d", transport.deletes)
	}
	listed, err := repo.ListCategories(ctx, category.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range listed {
		if c.ID == created.ID && c.ImageURL != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("list missing image")
	}
	_, err = db.Exec(ctx, `INSERT INTO product_category_map VALUES (1,$1)`, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	cats, err := repo.GetProductCategories(ctx, 1)
	if err != nil || len(cats) != 1 || cats[0].ImageURL == nil {
		t.Fatal("product category projection missing image", err)
	}
	fetched := request("GET", fmt.Sprintf("/categories/%d", created.ID), "", nil, 200)
	if fetched.ImageURL == nil {
		t.Fatal("detail API omitted image")
	}
}
