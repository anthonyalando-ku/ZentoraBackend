package tag

import (
	"strings"
	"unicode/utf8"
)

type CreateRequest struct {
	Name string `json:"name"`
}

func (r *CreateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 100 {
		return ErrInvalidName
	}
	return nil
}