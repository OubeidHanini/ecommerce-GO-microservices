// Load test: a burst of requests through the gateway, then a consistency check.
//
//   docker compose --profile loadtest run --rm k6
//   docker compose --profile loadtest run --rm -e REQUESTS=20000 -e VUS=100 k6
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE = __ENV.GATEWAY_URL || 'http://localhost:8080';
const REQUESTS = Number(__ENV.REQUESTS || 100000);
const VUS = Number(__ENV.VUS || 200);
const INITIAL_STOCK = 1000000;
const JSON_HEADERS = { 'Content-Type': 'application/json' };

export const options = {
  scenarios: {
    burst: { executor: 'shared-iterations', vus: VUS, iterations: REQUESTS, maxDuration: '15m' },
  },
  // The run fails if more than 1% of requests fail or if p95 latency exceeds 500 ms.
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
  summaryTrendStats: ['avg', 'p(50)', 'p(95)', 'p(99)', 'max'],
};

export function setup() {
  const user = http.post(`${BASE}/users`,
    JSON.stringify({ name: 'Load test', email: `load-${Date.now()}@example.com` }), { headers: JSON_HEADERS });
  const product = http.post(`${BASE}/products`,
    JSON.stringify({ name: 'Load test item', price_cents: 1000, stock: INITIAL_STOCK }), { headers: JSON_HEADERS });
  if (user.status !== 201 || product.status !== 201) {
    throw new Error(`setup failed: user ${user.status}, product ${product.status}`);
  }
  return { userId: user.json('id'), productId: product.json('id') };
}

export default function (data) {
  // 90% reads, 10% orders: in a shop, browsing dwarfs buying.
  if (Math.random() < 0.9) {
    const res = http.get(`${BASE}/products/${data.productId}`, { tags: { name: 'GET /products/{id}' } });
    check(res, { 'product: 200': (r) => r.status === 200 });
  } else {
    const res = http.post(`${BASE}/orders`,
      JSON.stringify({ user_id: data.userId, items: [{ product_id: data.productId, quantity: 1 }] }),
      { headers: JSON_HEADERS, tags: { name: 'POST /orders' } });
    check(res, { 'order: 202': (r) => r.status === 202 });
  }
}

// After the burst, every accepted order must be confirmed and the stock must have
// dropped by exactly that many units: no lost event, no double reservation.
export function teardown(data) {
  let confirmed = 0;
  let pending = 0;
  for (let attempt = 0; attempt < 30; attempt++) {
    const orders = http.get(`${BASE}/orders`, { timeout: '60s' }).json().filter((o) => o.user_id === data.userId);
    confirmed = orders.filter((o) => o.status === 'confirmed').length;
    pending = orders.filter((o) => o.status === 'pending').length;
    if (pending === 0) break;
    sleep(2);
  }
  const stock = http.get(`${BASE}/products/${data.productId}`).json('stock');

  console.log(`orders confirmed: ${confirmed}, still pending: ${pending}`);
  console.log(`stock: ${INITIAL_STOCK} -> ${stock} (expected ${INITIAL_STOCK - confirmed})`);
  if (pending !== 0 || stock !== INITIAL_STOCK - confirmed) {
    throw new Error('consistency check FAILED');
  }
  console.log('consistency check PASSED: stock matches confirmed orders exactly');
}
