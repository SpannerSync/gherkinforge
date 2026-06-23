package usecases

import (
	"context"
	"fmt"

	"github.com/spannersync/gherkinforge/pkg/context/order/domain"
)

// CreateOrderCommand carries the intent from the transport layer.
type CreateOrderCommand struct {
	OrderID    string
	CustomerID string
	Items      []CommandItem
}

// CommandItem is a line item within CreateOrderCommand.
type CommandItem struct {
	ProductID string
	Quantity  int
}

// CreateOrderResult is returned on success.
type CreateOrderResult struct {
	Order domain.Order
}

// CreateOrderUseCase orchestrates order creation against domain ports.
type CreateOrderUseCase struct {
	repo      domain.OrderRepository
	publisher domain.EventPublisher
	inventory domain.Inventory
}

// NewCreateOrderUseCase constructs the use case with its required ports.
func NewCreateOrderUseCase(
	repo domain.OrderRepository,
	publisher domain.EventPublisher,
	inventory domain.Inventory,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		repo:      repo,
		publisher: publisher,
		inventory: inventory,
	}
}

// Execute runs the CreateOrder business workflow.
func (uc *CreateOrderUseCase) Execute(ctx context.Context, cmd CreateOrderCommand) (CreateOrderResult, error) {
	items := make([]domain.OrderItem, 0, len(cmd.Items))
	for _, ci := range cmd.Items {
		product, err := uc.inventory.FindProduct(ctx, ci.ProductID)
		if err != nil {
			return CreateOrderResult{}, fmt.Errorf("resolving product %s: %w", ci.ProductID, err)
		}
		items = append(items, domain.OrderItem{
			ProductID: ci.ProductID,
			Quantity:  ci.Quantity,
			UnitPence: product.UnitPence,
		})
	}

	order, err := domain.NewOrder(cmd.OrderID, cmd.CustomerID, items)
	if err != nil {
		return CreateOrderResult{}, fmt.Errorf("creating order: %w", err)
	}

	if err = uc.repo.Save(ctx, order); err != nil {
		return CreateOrderResult{}, fmt.Errorf("saving order: %w", err)
	}

	if err = uc.publisher.Publish(ctx, "order.created", map[string]any{
		"order_id":    order.ID,
		"customer_id": order.CustomerID,
		"total_pence": order.TotalPence,
	}); err != nil {
		return CreateOrderResult{}, fmt.Errorf("publishing event: %w", err)
	}

	return CreateOrderResult{Order: order}, nil
}
