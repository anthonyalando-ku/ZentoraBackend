package catalog

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"net/http"
	"zentora-service/internal/domain/category"
)

// Existing JSON remains supported. Uploads carry JSON in `data` and one `image`.
func bindCategoryRequest(c *gin.Context, target any) (*multipart.FileHeader, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 9<<20)
	if c.ContentType() != "multipart/form-data" {
		return nil, c.ShouldBindJSON(target)
	}
	if err := c.Request.ParseMultipartForm(9 << 20); err != nil {
		return nil, err
	}
	form := c.Request.MultipartForm
	if form != nil {
		defer form.RemoveAll()
	}
	// The body limit is below the in-memory threshold: returned files remain in memory.
	if len(form.Value["data"]) != 1 {
		return nil, fmt.Errorf("multipart data must contain category JSON")
	}
	if err := json.Unmarshal([]byte(form.Value["data"][0]), target); err != nil {
		return nil, err
	}
	files := form.File["image"]
	if len(files) != 1 || len(form.File) != 1 || files[0].Size > 8<<20 {
		return nil, category.ErrInvalidImage
	}
	return files[0], nil
}

func createCategoryWithFile(h *CatalogHandler, c *gin.Context, req *category.CreateRequest, file *multipart.FileHeader) (*category.Category, error) {
	if file == nil {
		return h.svc.CreateCategory(c.Request.Context(), req)
	}
	return h.svc.CreateCategory(c.Request.Context(), req, file)
}
func updateCategoryWithFile(h *CatalogHandler, c *gin.Context, id int64, req *category.UpdateRequest, file *multipart.FileHeader) (*category.Category, error) {
	if file == nil {
		return h.svc.UpdateCategory(c.Request.Context(), id, req)
	}
	return h.svc.UpdateCategory(c.Request.Context(), id, req, file)
}
