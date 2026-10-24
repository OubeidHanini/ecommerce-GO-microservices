package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/postgres"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/server"
	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/httpapi"
	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/storage"
	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/user"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run is the only place that knows every concrete implementation.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()

	repo := storage.NewUserRepository(db)
	if err := repo.Migrate(ctx); err != nil {
		return err
	}

	handler := httpapi.NewHandler(user.NewService(repo))
	return server.Run(ctx, server.New(":"+server.Getenv("PORT", "8081"), handler))
}
