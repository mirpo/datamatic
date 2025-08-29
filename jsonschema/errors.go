package jsonschema

import "errors"

var (
	ErrInvalidJSON          = errors.New("invalid JSON")
	ErrMissingRequiredField = errors.New("missing required field")
	ErrInvalidFieldType     = errors.New("invalid type for field")
	ErrUnknownProperty      = errors.New("unknown property")
	ErrValueNotInEnum       = errors.New("value is not in enum")
	ErrExpectedString       = errors.New("expected string")
	ErrExpectedNumber       = errors.New("expected number")
	ErrExpectedInteger      = errors.New("expected integer")
	ErrExpectedBoolean      = errors.New("expected boolean")
	ErrExpectedArray        = errors.New("expected array")
	ErrExpectedObject       = errors.New("expected object")
	ErrUnsupportedType      = errors.New("unsupported type")
)
