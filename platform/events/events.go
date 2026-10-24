// Package events defines the contracts of messages exchanged between services.
// Sharing them keeps the producer and the consumer from silently drifting apart.
package events

// Routing keys.
const (
	OrderCreated  = "order.created"
	StockReserved = "stock.reserved"
	StockRejected = "stock.rejected"
)

type OrderItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type OrderCreatedEvent struct {
	OrderID int64       `json:"order_id"`
	Items   []OrderItem `json:"items"`
}

// StockResultEvent is published with the StockReserved or StockRejected key.
type StockResultEvent struct {
	OrderID int64  `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}
