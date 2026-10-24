// The gateway is the only exposed entry point: clients don't need to know each
// service's address, and the services stay on the internal network.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/httpx"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	router, err := newRouter(map[string]string{
		"users":    os.Getenv("USER_SERVICE_URL"),
		"products": os.Getenv("PRODUCT_SERVICE_URL"),
		"orders":   os.Getenv("ORDER_SERVICE_URL"),
	})
	if err != nil {
		return err
	}
	return server.Run(ctx, server.New(":"+server.Getenv("PORT", "8080"), router))
}

// newRouter maps each resource (/users, /users/…) to the service that owns it.
func newRouter(upstreams map[string]string) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health)

	for resource, rawURL := range upstreams {
		target, err := url.Parse(rawURL)
		if err != nil || target.Host == "" {
			return nil, fmt.Errorf("invalid upstream URL for %s: %q", resource, rawURL)
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("%s upstream: %v", resource, err)
			httpx.JSON(w, http.StatusBadGateway, map[string]string{"error": resource + " service unavailable"})
		}
		mux.Handle("/"+resource, proxy)
		mux.Handle("/"+resource+"/", proxy)
	}
	return mux, nil
}
