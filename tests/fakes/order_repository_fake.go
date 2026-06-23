package fakes

import (
	"context"
	"sync"

	"github.com/spannersync/gherkinforge/pkg/context/order/domain"
)

// OrderRepositoryFake is a hand-written test double that records calls.
// It implements domain.OrderRepository without any infrastructure dependency.
type OrderRepositoryFake struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
	SaveFn func(ctx context.Context, order domain.Order) error // optional override
}

// NewOrderRepositoryFake returns an empty fake.
func NewOrderRepositoryFake() *OrderRepositoryFake {
	return &OrderRepositoryFake{orders: make(map[string]domain.Order)}
}

// Save stores the order. Delegates to SaveFn if set, otherwise uses default map storage.
func (f *OrderRepositoryFake) Save(ctx context.Context, order domain.Order) error {
	if f.SaveFn != nil {
		return f.SaveFn(ctx, order)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orders[order.ID] = order
	return nil
}

// FindByID returns the stored order or domain.ErrOrderNotFound.
func (f *OrderRepositoryFake) FindByID(_ context.Context, id string) (domain.Order, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	o, ok := f.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return o, nil
}

// All returns every stored order (useful for assertions in tests).
func (f *OrderRepositoryFake) All() []domain.Order {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]domain.Order, 0, len(f.orders))
	for _, o := range f.orders {
		out = append(out, o)
	}
	return out
}
