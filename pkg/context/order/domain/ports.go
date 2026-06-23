package domain

import "context"

// OrderRepository is the port that infrastructure adapters must implement.
// Domain code depends on this interface, never on concrete DB drivers.
type OrderRepository interface {
	Save(ctx context.Context, order Order) error
	FindByID(ctx context.Context, id string) (Order, error)
}

// EventPublisher is the port for emitting domain events to a broker.
// Concrete implementations (Kafka, NATS, in-memory) live in adapters/.
type EventPublisher interface {
	Publish(ctx context.Context, eventName string, payload any) error
}

// Inventory is the port for querying product catalogue data.
type Inventory interface {
	FindProduct(ctx context.Context, productID string) (Product, error)
}

// Product is a read-model returned by the Inventory port.
type Product struct {
	ID        string
	SKU       string
	UnitPence int64
	Stock     int
}
