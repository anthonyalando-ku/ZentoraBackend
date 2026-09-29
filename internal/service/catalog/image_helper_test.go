package catalog

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestProductImageCompressionAndLocalFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		t.Fatal(err)
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 1200, 600))); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	f, _ := w.CreateFormFile("images", "product.png")
	_, _ = f.Write(raw.Bytes())
	_ = w.Close()
	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	defer req.MultipartForm.RemoveAll()
	path, err := processSingleImage(req.MultipartForm.File["images"][0], nil)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer saved.Close()
	data, _ := io.ReadAll(saved)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "webp" || cfg.Width != 800 || cfg.Height != 400 || !strings.HasSuffix(path, ".webp") {
		t.Fatalf("product pipeline changed: %+v %s %v", cfg, format, err)
	}
}
