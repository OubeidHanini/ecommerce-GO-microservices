package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/config"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/events"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/outbox"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/postgres"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/rabbitmq"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/server"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/httpapi"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/messaging"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/product"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/storage"
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
	port := env.Get("PORT", "8082")
	dbCfg := postgres.ConfigFromEnv(env)
	mqCfg := rabbitmq.ConfigFromEnv(env)
	if err := env.Err(); err != nil {
		return err
	}

	db, err := postgres.Connect(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer db.Close()

	repo := storage.NewProductRepository(db)
	if err := repo.Migrate(ctx); err != nil {
		return err
	}

	broker, err := rabbitmq.Connect(ctx, mqCfg)
	if err != nil {
		return err
	}
	defer broker.Close()
	broker.CancelOnConnectionLost(cancel)

	products := product.NewService(repo)
	err = broker.Subscribe(ctx, "product-service.order-created", []string{events.OrderCreated},
		messaging.OrderCreated(products))
	if err != nil {
		return err
	}
	go outbox.Relay(ctx, db, broker)

	err = server.Run(ctx, server.New(":"+port, httpapi.NewHandler(products)))
	if cause := context.Cause(ctx); errors.Is(cause, rabbitmq.ErrConnectionLost) {
		return cause
	}
	return err
}
