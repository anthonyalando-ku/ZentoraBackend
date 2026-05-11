package catalog

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	imagekit "github.com/imagekit-developer/imagekit-go/v2"
	"golang.org/x/sync/errgroup"
)

const (
	uploadDir      = "uploads/products"
	imagekitFolder = "/zentora"
	maxImageWidth  = 800
	webpQuality    = 75

	// Maximum images processed in parallel.
	// Keeps memory bounded (each image can be several MB in-flight)
	// and avoids hammering the ImageKit API with too many concurrent requests.
	maxConcurrent = 4
)

// saveImages processes all files concurrently up to maxConcurrent at a time.
// Results are returned in the same order as the input slice.
func saveImages(files []*multipart.FileHeader, ik *imagekit.Client) ([]string, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	// Pre-allocate result slice — each goroutine writes to its own index,
	// so no mutex is needed on the slice itself.
	paths := make([]string, len(files))

	// errgroup cancels all goroutines as soon as one returns an error.
	g, ctx := errgroup.WithContext(context.Background())

	// Semaphore channel caps parallelism without a worker pool.
	sem := make(chan struct{}, maxConcurrent)

loop:
	for i, fh := range files {
		i, fh := i, fh // capture loop variables

		select {
		case sem <- struct{}{}:
			// slot acquired — launch goroutine
		case <-ctx.Done():
			// a previous goroutine already failed; stop queueing
			break loop
		}

		g.Go(func() error {
			defer func() { <-sem }()

			path, err := processSingleImage(fh, ik)
			if err != nil {
				return fmt.Errorf("image %d (%q): %w", i+1, fh.Filename, err)
			}
			paths[i] = path
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return paths, nil
}

// ── The rest of the file is unchanged ────────────────────────────────────────

func processSingleImage(fh *multipart.FileHeader, ik *imagekit.Client) (string, error) {
	raw, err := readFileHeader(fh)
	if err != nil {
		return "", err
	}

	compressed, err := compressToWebP(raw)
	if err != nil {
		return "", err
	}

	filename := buildFilename(fh.Filename)

	if ik != nil {
		if url, ok := uploadToImageKit(ik, filename, compressed); ok {
			return url, nil
		}
	}

	return saveLocally(filename, compressed)
}

func readFileHeader(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", fh.Filename, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return nil, fmt.Errorf("read %q: %w", fh.Filename, err)
	}
	return buf.Bytes(), nil
}

func compressToWebP(raw []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	resized := imaging.Resize(img, maxImageWidth, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, resized, &webp.Options{Quality: float32(webpQuality)}); err != nil {
		return nil, fmt.Errorf("webp encode: %w", err)
	}

	return buf.Bytes(), nil
}

func uploadToImageKit(ik *imagekit.Client, filename string, data []byte) (string, bool) {
	resp, err := ik.Files.Upload(
		context.Background(),
		imagekit.FileUploadParams{
			File:     bytes.NewReader(data),
			FileName: filename,
			Folder:   imagekit.String(imagekitFolder),
		},
	)
	if err != nil || resp.URL == "" {
		return "", false
	}
	return resp.URL, true
}

func saveLocally(filename string, data []byte) (string, error) {
	dest := filepath.Join(uploadDir, filename)
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return "", fmt.Errorf("write %q: %w", dest, err)
	}
	return dest, nil
}

func buildFilename(original string) string {
	base := filepath.Base(original)
	ext := filepath.Ext(base)
	name := strings.ReplaceAll(strings.TrimSuffix(base, ext), " ", "_")
	return fmt.Sprintf("%d_%s.webp", time.Now().UnixNano(), name)
}

// buildFilenameWithIndex appends the slot index to guarantee uniqueness when
// multiple images are processed in the same nanosecond (common in tests).
// Use this instead of buildFilename if you observe collisions.
var filenameOnce sync.Mutex

func buildFilenameUnique(original string, idx int) string {
	base := filepath.Base(original)
	ext := filepath.Ext(base)
	name := strings.ReplaceAll(strings.TrimSuffix(base, ext), " ", "_")

	filenameOnce.Lock()
	ts := time.Now().UnixNano()
	filenameOnce.Unlock()

	return fmt.Sprintf("%d_%d_%s.webp", ts, idx, name)
}