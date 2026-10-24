package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/httpapi"
	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/postgres"
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

	repo := postgres.NewUserRepository(db)
	if err := repo.Migrate(ctx); err != nil {
		return err
	}

	server := &http.Server{
		Addr:              ":" + getenv("PORT", "8081"),
		Handler:           httpapi.NewHandler(user.NewService(repo)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return serve(ctx, server)
}

func serve(ctx context.Context, server *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("user-service listening on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
