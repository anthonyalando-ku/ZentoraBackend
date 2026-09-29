package catalog

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"log"
	"mime/multipart"
	"time"
	"zentora-service/internal/domain/category"
)

func (s *CatalogService) uploadCategoryImage(ctx context.Context, file *multipart.FileHeader) (*string, func(), error) {
	noop := func() {}
	if file == nil || file.Size <= 0 || file.Size > 8<<20 {
		return nil, noop, category.ErrInvalidImage
	}
	f, err := file.Open()
	if err != nil {
		return nil, noop, category.ErrInvalidImage
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (8<<20)+1))
	if err != nil || len(raw) > 8<<20 {
		return nil, noop, category.ErrInvalidImage
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png" && format != "gif" && format != "webp") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16000000 || cfg.Height > 10*cfg.Width || cfg.Width > 10*cfg.Height {
		return nil, noop, category.ErrInvalidImage
	}
	// Exactly the same resize/WebP encoder used by product images.
	data, err := compressToWebP(raw)
	if err != nil {
		return nil, noop, category.ErrInvalidImage
	}
	if s.imageKit == nil {
		return nil, noop, category.ErrImageUpload
	}
	uploadCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	url, fileID, err := uploadImageKitFile(uploadCtx, s.imageKit, buildFilename(file.Filename), data)
	cleanup := func() {
		if fileID == "" {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.imageKit.Files.Delete(cleanupCtx, fileID); err != nil {
			log.Printf("category image cleanup failed for %s: %v", fileID, err)
		}
	}
	if err != nil {
		cleanup()
		return nil, noop, category.ErrImageUpload
	}
	normalized, err := category.NormalizeImageURL(&url)
	if err != nil || normalized == nil {
		cleanup()
		return nil, noop, fmt.Errorf("%w: invalid uploaded URL", category.ErrImageUpload)
	}
	return normalized, cleanup, nil
}
