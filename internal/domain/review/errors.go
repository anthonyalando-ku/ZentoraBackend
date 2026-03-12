package review

import "errors"

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrForbidden        = errors.New("forbidden")
	ErrReviewWindowEnded = errors.New("review window ended")
	ErrOrderNotCompleted = errors.New("order not completed")
)