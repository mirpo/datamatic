package step

import "errors"

var (
	ErrUnsupportedStepType = errors.New("unsupported step type")
	ErrStepNotFound        = errors.New("step not found")
	ErrKeyNotFound         = errors.New("key not found")
)