package domain

import "errors"

var (
	ErrNoItems         = errors.New("order must contain at least one item")
	ErrInvalidQuantity = errors.New("item quantity must be positive")
	ErrOrderNotFound   = errors.New("order not found")
	ErrProductNotFound = errors.New("product not found in inventory")
)
