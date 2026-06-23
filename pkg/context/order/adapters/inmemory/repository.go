package inmemory

import (
	"context"
	"sync"

	"github.com/spannersync/gherkinforge/pkg/context/order/domain"
)

// OrderRepository is a thread-safe in-memory implementation of domain.OrderRepository.
// Use this in unit tests and local demos — not in production.
type OrderRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

// NewOrderRepository returns an empty in-memory store.
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{orders: make(map[string]domain.Order)}
}

// Save stores the order by its ID, overwriting any previous entry.
func (r *OrderRepository) Save(_ context.Context, order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

// FindByID returns the order for the given ID or domain.ErrOrderNotFound.
func (r *OrderRepository) FindByID(_ context.Context, id string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return o, nil
}

// InventoryStore is an in-memory implementation of domain.Inventory.
type InventoryStore struct {
	mu       sync.RWMutex
	products map[string]domain.Product
}

// NewInventoryStore returns an empty inventory; seed it with Seed().
func NewInventoryStore() *InventoryStore {
	return &InventoryStore{products: make(map[string]domain.Product)}
}

// Seed populates the inventory from a slice of products.
func (s *InventoryStore) Seed(products []domain.Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range products {
		s.products[p.ID] = p
	}
}

// FindProduct returns the product or domain.ErrProductNotFound.
func (s *InventoryStore) FindProduct(_ context.Context, productID string) (domain.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[productID]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return p, nil
}
