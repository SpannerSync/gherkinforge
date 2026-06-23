package fakes

import (
	"context"
	"sync"

	"github.com/spannersync/gherkinforge/pkg/context/order/domain"
)

// InventoryFake is a hand-written test double for domain.Inventory.
type InventoryFake struct {
	mu       sync.RWMutex
	products map[string]domain.Product
}

// NewInventoryFake returns an empty fake.
func NewInventoryFake() *InventoryFake {
	return &InventoryFake{products: make(map[string]domain.Product)}
}

// Seed loads products into the fake so step definitions can prime the context.
func (f *InventoryFake) Seed(products []domain.Product) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range products {
		f.products[p.ID] = p
	}
}

// FindProduct returns the product or domain.ErrProductNotFound.
func (f *InventoryFake) FindProduct(_ context.Context, productID string) (domain.Product, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	p, ok := f.products[productID]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return p, nil
}
