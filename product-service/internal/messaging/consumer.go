// Package messaging turns RabbitMQ messages into domain calls.
package messaging

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/events"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/rabbitmq"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/product"
)

func OrderCreated(products *product.Service) rabbitmq.Handler {
	return func(ctx context.Context, body []byte) error {
		var evt events.OrderCreatedEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			return rabbitmq.Permanent(err)
		}

		lines := make([]product.Line, len(evt.Items))
		for i, item := range evt.Items {
			lines[i] = product.Line{ProductID: item.ProductID, Quantity: item.Quantity}
		}

		err := products.Reserve(ctx, evt.OrderID, lines)
		if errors.Is(err, product.ErrInvalidInput) {
			return rabbitmq.Permanent(err)
		}
		return err
	}
}
