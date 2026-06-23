// Package app wires the transport layer (HTTP) to the use cases.
// It is the outermost ring of the hexagonal architecture — it imports
// everything but is imported by nothing except cmd/.
package app

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spannersync/gherkinforge/pkg/context/order/adapters/inmemory"
	"github.com/spannersync/gherkinforge/pkg/context/order/usecases"
)

// Server holds the HTTP router and wired use cases.
type Server struct {
	router *gin.Engine
	uc     *usecases.CreateOrderUseCase
}

// New constructs a Server with all dependencies wired using in-memory adapters.
// Replace inmemory adapters with real DB adapters for production use.
func New() *Server {
	repo := inmemory.NewOrderRepository()
	inv := inmemory.NewInventoryStore()
	pub := &noopPublisher{}
	uc := usecases.NewCreateOrderUseCase(repo, pub, inv)

	r := gin.Default()
	s := &Server{router: r, uc: uc}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.POST("/orders", s.handleCreateOrder)
	s.router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func (s *Server) handleCreateOrder(c *gin.Context) {
	var body struct {
		CustomerID string `json:"customer_id" binding:"required"`
		Items      []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items := make([]usecases.CommandItem, len(body.Items))
	for i, it := range body.Items {
		items[i] = usecases.CommandItem{ProductID: it.ProductID, Quantity: it.Quantity}
	}

	result, err := s.uc.Execute(c.Request.Context(), usecases.CreateOrderCommand{
		OrderID:    "ORD-" + body.CustomerID,
		CustomerID: body.CustomerID,
		Items:      items,
	})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"order_id":    result.Order.ID,
		"total_pence": result.Order.TotalPence,
	})
}

// Handler returns the underlying http.Handler (used in tests).
func (s *Server) Handler() http.Handler { return s.router }

// noopPublisher silences event publishing in the HTTP stub.
type noopPublisher struct{}

func (*noopPublisher) Publish(_ context.Context, _ string, _ any) error { return nil }
