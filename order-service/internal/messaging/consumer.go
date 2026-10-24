// Package messaging turns RabbitMQ messages into domain calls.
package messaging

import (
	"context"
	"encoding/json"

	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/order"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/events"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/rabbitmq"
)

func StockReserved(orders *order.Service) rabbitmq.Handler {
	return func(ctx context.Context, body []byte) error {
		evt, err := decode(body)
		if err != nil {
			return err
		}
		return orders.Confirm(ctx, evt.OrderID)
	}
}

func StockRejected(orders *order.Service) rabbitmq.Handler {
	return func(ctx context.Context, body []byte) error {
		evt, err := decode(body)
		if err != nil {
			return err
		}
		return orders.Cancel(ctx, evt.OrderID, evt.Reason)
	}
}

func decode(body []byte) (events.StockResultEvent, error) {
	var evt events.StockResultEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return evt, rabbitmq.Permanent(err)
	}
	return evt, nil
}
