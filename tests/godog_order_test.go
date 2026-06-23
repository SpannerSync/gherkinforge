package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/spannersync/gherkinforge/pkg/context/order/domain"
	"github.com/spannersync/gherkinforge/pkg/context/order/usecases"
	"github.com/spannersync/gherkinforge/tests/fakes"
)

// scenarioState holds per-scenario collaborators and captured results.
type scenarioState struct {
	inventory *fakes.InventoryFake
	repo      *fakes.OrderRepositoryFake
	publisher *fakes.EventPublisherFake
	useCase   *usecases.CreateOrderUseCase
	lastOrder domain.Order
	lastErr   error
}

type ctxKey struct{}

func newState() *scenarioState {
	inv := fakes.NewInventoryFake()
	repo := fakes.NewOrderRepositoryFake()
	pub := fakes.NewEventPublisherFake()
	return &scenarioState{
		inventory: inv,
		repo:      repo,
		publisher: pub,
		useCase:   usecases.NewCreateOrderUseCase(repo, pub, inv),
	}
}

func stateFrom(ctx context.Context) *scenarioState {
	return ctx.Value(ctxKey{}).(*scenarioState)
}

// ── Background step ──────────────────────────────────────────────────────────

func theSystemContainsTheFollowingValidProductInventory(ctx context.Context, table *godog.Table) (context.Context, error) {
	s := stateFrom(ctx)
	for _, row := range table.Rows[1:] { // skip header
		unitPence, err := strconv.ParseInt(row.Cells[2].Value, 10, 64)
		if err != nil {
			return ctx, fmt.Errorf("parsing unit_pence for row %v: %w", row.Cells, err)
		}
		stock, err := strconv.Atoi(row.Cells[3].Value)
		if err != nil {
			return ctx, fmt.Errorf("parsing stock for row %v: %w", row.Cells, err)
		}
		s.inventory.Seed([]domain.Product{{
			ID:        row.Cells[0].Value,
			SKU:       row.Cells[1].Value,
			UnitPence: unitPence,
			Stock:     stock,
		}})
	}
	return ctx, nil
}

// ── Given steps ──────────────────────────────────────────────────────────────

func aCustomerAggregateInitializedWithID(ctx context.Context, customerID string) (context.Context, error) {
	s := stateFrom(ctx)
	s.lastOrder = domain.Order{CustomerID: customerID}
	return ctx, nil
}

// ── When steps ───────────────────────────────────────────────────────────────

type createOrderPayload struct {
	CustomerID string `json:"customer_id"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

func theCustomerSubmitsACreateOrderCommandWithTheFollowingPayload(ctx context.Context, doc *godog.DocString) (context.Context, error) {
	s := stateFrom(ctx)
	var p createOrderPayload
	if err := json.Unmarshal([]byte(doc.Content), &p); err != nil {
		return ctx, fmt.Errorf("parsing command payload: %w", err)
	}

	items := make([]usecases.CommandItem, len(p.Items))
	for i, it := range p.Items {
		items[i] = usecases.CommandItem{ProductID: it.ProductID, Quantity: it.Quantity}
	}

	result, err := s.useCase.Execute(ctx, usecases.CreateOrderCommand{
		OrderID:    "ORD-" + p.CustomerID,
		CustomerID: p.CustomerID,
		Items:      items,
	})
	s.lastErr = err
	if err == nil {
		s.lastOrder = result.Order
	}
	return ctx, nil
}

// ── Then steps ───────────────────────────────────────────────────────────────

func theOrderAggregateShouldBeSuccessfullyCreated(ctx context.Context) error {
	s := stateFrom(ctx)
	if s.lastErr != nil {
		return fmt.Errorf("expected order creation to succeed, got: %w", s.lastErr)
	}
	if s.lastOrder.ID == "" {
		return fmt.Errorf("order ID must not be empty")
	}
	return nil
}

func anDomainEventIsPublishedToTheBroker(ctx context.Context, eventName string) error {
	s := stateFrom(ctx)
	if !s.publisher.HasEvent(eventName) {
		return fmt.Errorf("expected event %q to be published; got %v", eventName, s.publisher.Published())
	}
	return nil
}

func theTotalOrderValueInPenceShouldBe(ctx context.Context, expected int64) error {
	s := stateFrom(ctx)
	if s.lastOrder.TotalPence != expected {
		return fmt.Errorf("expected TotalPence %d, got %d", expected, s.lastOrder.TotalPence)
	}
	return nil
}

func theOrderCreationShouldFailWith(ctx context.Context, errMsg string) error {
	s := stateFrom(ctx)
	if s.lastErr == nil {
		return fmt.Errorf("expected order creation to fail with %q, but it succeeded", errMsg)
	}
	if !strings.Contains(s.lastErr.Error(), errMsg) {
		return fmt.Errorf("expected error containing %q, got %q", errMsg, s.lastErr.Error())
	}
	return nil
}

// Zero Trust Pillar 4 — mutation-proof assertion: ID must be non-empty.
// A mutant that clears the generated ID will fail this step.
func theOrderIDShouldNotBeEmpty(ctx context.Context) error {
	s := stateFrom(ctx)
	if s.lastOrder.ID == "" {
		return fmt.Errorf("ZERO TRUST: order ID is empty — mutant detected or aggregate construction failed")
	}
	return nil
}

// ── Suite wiring ─────────────────────────────────────────────────────────────

func InitializeScenario(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
		return context.WithValue(ctx, ctxKey{}, newState()), nil
	})

	sc.Step(`^the system contains the following valid product inventory:$`, theSystemContainsTheFollowingValidProductInventory)
	sc.Step(`^a customer aggregate initialized with ID "([^"]*)"$`, aCustomerAggregateInitializedWithID)
	sc.Step(`^the customer submits a create order command with the following payload:$`, theCustomerSubmitsACreateOrderCommandWithTheFollowingPayload)
	sc.Step(`^the order aggregate should be successfully created$`, theOrderAggregateShouldBeSuccessfullyCreated)
	sc.Step(`^an "([^"]*)" domain event is published to the broker$`, anDomainEventIsPublishedToTheBroker)
	sc.Step(`^the total order value in pence should be (\d+)$`, theTotalOrderValueInPenceShouldBe)
	sc.Step(`^the order creation should fail with "([^"]*)"$`, theOrderCreationShouldFailWith)
	sc.Step(`^the order ID should not be empty$`, theOrderIDShouldNotBeEmpty)
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/business"},
			TestingT: t,
			Strict:   true,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature tests failed")
	}
}
