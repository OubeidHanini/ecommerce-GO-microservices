// Package clients calls the other services over HTTP.
// It implements the order.Users and order.Catalog interfaces.
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Without a timeout, one slow service would block every order in flight.
var httpClient = &http.Client{Timeout: 3 * time.Second}

type UserClient struct {
	baseURL string
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *UserClient) Exists(ctx context.Context, id int64) (bool, error) {
	return getJSON(ctx, fmt.Sprintf("%s/users/%d", c.baseURL, id), nil)
}

type ProductClient struct {
	baseURL string
}

func NewProductClient(baseURL string) *ProductClient {
	return &ProductClient{baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *ProductClient) Price(ctx context.Context, productID int64) (int64, bool, error) {
	var p struct {
		PriceCents int64 `json:"price_cents"`
	}
	found, err := getJSON(ctx, fmt.Sprintf("%s/products/%d", c.baseURL, productID), &p)
	return p.PriceCents, found, err
}

// getJSON returns found=false on a 404; dst may be nil when only existence matters.
func getJSON(ctx context.Context, url string, dst any) (found bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if dst == nil {
			return true, nil
		}
		return true, json.NewDecoder(resp.Body).Decode(dst)
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}
}
