# ecommerce-GO-microservices

A small e-commerce platform built as Go microservices: REST between services, events over RabbitMQ, one PostgreSQL database per service.

```
client ──► gateway :8080 ──► user-service    :8081 ─► users_db
                        ├──► product-service :8082 ─► products_db
                        └──► order-service   :8083 ─► orders_db
                                   │  ▲
                 order.created     ▼  │ stock.reserved / stock.rejected
                              RabbitMQ (exchange "ecommerce.events")
```

## Order lifecycle

1. `POST /orders`: the order-service checks the user and reads prices over REST, then stores the order as `pending` (response `202`).
2. The product-service consumes `order.created` and reserves stock for every line, or for none.
3. The order-service consumes the result and moves the order to `confirmed` or `cancelled` (with a `reason`).

## Design choices and the problem each one solves

| Choice | Problem solved |
|---|---|
| Layers `domain → storage / httpapi / messaging`, interfaces declared in the domain | Business logic depends on neither PostgreSQL nor HTTP, so it is tested with fakes (`order_test.go`) |
| Transactional outbox (`platform/outbox`) | A crash between COMMIT and publish would lose the event and leave the order `pending` forever |
| `processed_orders` table and `UPDATE … WHERE status = 'pending'` | RabbitMQ delivers at least once: a replayed message must not take stock twice |
| `UPDATE … WHERE stock >= quantity` | Two concurrent orders cannot sell the same item |
| Prices in integer cents, frozen in the order | No rounding errors; a price change does not alter past orders |
| Shared `platform` module | Database, broker, HTTP server and event contracts are written once |

## Run

```sh
docker compose up -d --build
# if port 8080 is already taken: GATEWAY_PORT=8090 docker compose up -d --build
```

## Try it

```sh
curl -s -X POST localhost:8080/users    -d '{"name":"Ada","email":"ada@example.com"}'
curl -s -X POST localhost:8080/products -d '{"name":"Keyboard","price_cents":4990,"stock":3}'
curl -s -X POST localhost:8080/orders   -d '{"user_id":1,"items":[{"product_id":1,"quantity":2}]}'
curl -s localhost:8080/orders/1   # "status": "confirmed" after a moment
```

RabbitMQ management UI: http://localhost:15672 (guest / guest).

## Development

```sh
go test ./order-service/...   # the root go.work links the modules together
```
