package service

import (
	"errors"
	"fmt"
)

var (
	ErrValidation        = errors.New("validation error")
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("conflict")
	ErrInsufficientStock = errors.New("insufficient stock")
)

func Validation(message string) error {
	return fmt.Errorf("%w: %s", ErrValidation, message)
}

func NotFound(message string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, message)
}

func Conflict(message string) error {
	return fmt.Errorf("%w: %s", ErrConflict, message)
}

func InsufficientStock(message string) error {
	return fmt.Errorf("%w: %s", ErrInsufficientStock, message)
}
