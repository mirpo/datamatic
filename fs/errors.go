package fs

import "errors"

var (
	ErrEmptyFile          = errors.New("file is empty")
	ErrTargetLineNotFound = errors.New("target line not found")
	ErrEmptyImagePath     = errors.New("image path is empty")
	ErrNoMatchingFiles    = errors.New("no files matched pattern")
)
