package category

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

var ErrInvalidImage = errors.New("category image must be a valid HTTP(S) URL or a supported image upload")
var ErrImageUpload = errors.New("category image upload failed; please try again")

// Nil/empty means no image. URLs are stored, never fetched by the backend.
func NormalizeImageURL(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*value)
	if s == "" {
		return nil, nil
	}
	if len(s) > 2048 || strings.ContainsFunc(s, unicode.IsSpace) {
		return nil, ErrInvalidImage
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return nil, ErrInvalidImage
	}
	return &s, nil
}
