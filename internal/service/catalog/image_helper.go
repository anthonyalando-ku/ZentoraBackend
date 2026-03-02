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
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	imagekit "github.com/imagekit-developer/imagekit-go/v2"
)

const (
	uploadDir      = "uploads/products"
	imagekitFolder = "/zentora"
	maxImageWidth  = 800
	webpQuality    = 75
)

// saveImages uploads each file to ImageKit when the client is available,
// falling back to local disk when ImageKit is nil or the upload fails.
func saveImages(files []*multipart.FileHeader, ik *imagekit.Client) ([]string, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	paths := make([]string, 0, len(files))
	for _, fh := range files {
		path, err := processSingleImage(fh, ik)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

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
    // Make sure you use the `Options` struct from chai2010/webp
    if err := webp.Encode(&buf, resized, &webp.Options{Quality: float32(webpQuality)}); err != nil {
        return nil, fmt.Errorf("webp encode: %w", err)
    }

    return buf.Bytes(), nil
}

// uploadToImageKit sends the compressed image bytes to ImageKit as an io.Reader.
// The SDK's File field accepts any io.Reader; bytes.NewReader satisfies that interface.
// Folder is param.Opt[string] in the v2 SDK — use the imagekit.String() helper.
// Returns the CDN URL and true on success, empty string and false on any failure.
func uploadToImageKit(ik *imagekit.Client, filename string, data []byte) (string, bool) {
	resp, err := ik.Files.Upload(
		context.Background(),
		imagekit.FileUploadParams{
			File:     bytes.NewReader(data), // io.Reader — required by v2 SDK
			FileName: filename,
			Folder:   imagekit.String(imagekitFolder), // param.Opt[string] wrapper
		},
	)
	if err != nil {
		return "", false
	}
	if resp.URL == "" {
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
