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

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/config"
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

	env := config.New()
	port := env.Get("PORT", "8080")
	upstreams := map[string]string{
		"users":    env.Required("USER_SERVICE_URL"),
		"products": env.Required("PRODUCT_SERVICE_URL"),
		"orders":   env.Required("ORDER_SERVICE_URL"),
	}
	if err := env.Err(); err != nil {
		return err
	}

	router, err := newRouter(upstreams)
	if err != nil {
		return err
	}
	return server.Run(ctx, server.New(":"+port, router))
}

// newRouter maps each resource (/users, /users/…) to the service that owns it.
func newRouter(upstreams map[string]string) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health)
	transport := httpx.Transport()

	for resource, rawURL := range upstreams {
		target, err := url.Parse(rawURL)
		if err != nil || target.Host == "" {
			return nil, fmt.Errorf("invalid upstream URL for %s: %q", resource, rawURL)
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = transport
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("%s upstream: %v", resource, err)
			httpx.JSON(w, http.StatusBadGateway, map[string]string{"error": resource + " service unavailable"})
		}
		mux.Handle("/"+resource, proxy)
		mux.Handle("/"+resource+"/", proxy)
	}
	return mux, nil
}
