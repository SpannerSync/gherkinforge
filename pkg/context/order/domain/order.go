package domain

import (
	"time"
)

// Order is the aggregate root for the order bounded context.
type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	TotalPence int64 // monetary value in pence — never float64
	CreatedAt  time.Time
}

// OrderItem represents a single line within an Order.
type OrderItem struct {
	ProductID  string
	Quantity   int
	UnitPence  int64
}

// NewOrder constructs a valid Order aggregate and calculates TotalPence.
// Returns an error if items is empty or any quantity is non-positive.
func NewOrder(id, customerID string, items []OrderItem) (Order, error) {
	if len(items) == 0 {
		return Order{}, ErrNoItems
	}
	var total int64
	for _, it := range items {
		if it.Quantity <= 0 {
			return Order{}, ErrInvalidQuantity
		}
		total += it.UnitPence * int64(it.Quantity)
	}
	return Order{
		ID:         id,
		CustomerID: customerID,
		Items:      items,
		TotalPence: total,
		CreatedAt:  time.Now().UTC(),
	}, nil
}
