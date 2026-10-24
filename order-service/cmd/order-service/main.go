package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/clients"
	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/httpapi"
	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/messaging"
	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/order"
	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/storage"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/config"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/events"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/outbox"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/postgres"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/rabbitmq"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run is the only place that knows every concrete implementation.
func run() error {
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancelCause(sigCtx)
	defer cancel(nil)

	env := config.New()
	port := env.Get("PORT", "8083")
	dbCfg := postgres.ConfigFromEnv(env)
	mqCfg := rabbitmq.ConfigFromEnv(env)
	userServiceURL := env.Required("USER_SERVICE_URL")
	productServiceURL := env.Required("PRODUCT_SERVICE_URL")
	if err := env.Err(); err != nil {
		return err
	}

	db, err := postgres.Connect(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer db.Close()

	repo := storage.NewOrderRepository(db)
	if err := repo.Migrate(ctx); err != nil {
		return err
	}

	broker, err := rabbitmq.Connect(ctx, mqCfg)
	if err != nil {
		return err
	}
	defer broker.Close()
	broker.CancelOnConnectionLost(cancel)

	orders := order.NewService(repo,
		clients.NewUserClient(userServiceURL),
		clients.NewProductClient(productServiceURL),
	)

	subscriptions := []struct {
		queue, routingKey string
		handle            rabbitmq.Handler
	}{
		{"order-service.stock-reserved", events.StockReserved, messaging.StockReserved(orders)},
		{"order-service.stock-rejected", events.StockRejected, messaging.StockRejected(orders)},
	}
	for _, s := range subscriptions {
		if err := broker.Subscribe(ctx, s.queue, []string{s.routingKey}, s.handle); err != nil {
			return err
		}
	}
	go outbox.Relay(ctx, db, broker)

	err = server.Run(ctx, server.New(":"+port, httpapi.NewHandler(orders)))
	if cause := context.Cause(ctx); errors.Is(cause, rabbitmq.ErrConnectionLost) {
		return cause
	}
	return err
}
