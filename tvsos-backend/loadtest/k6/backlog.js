import http from 'k6/http';
import { check, sleep } from 'k6';
import { Gauge, Rate } from 'k6/metrics';

const baseURL = __ENV.BASE_URL || 'http://127.0.0.1:8088';
const rate = Number(__ENV.RATE || 1);
const duration = __ENV.DURATION || '45s';

const pendingShipments = new Gauge('pending_shipments');
const businessFailures = new Rate('business_failures');

export const options = {
  scenarios: {
    producer: {
      executor: 'constant-arrival-rate',
      exec: 'produce',
      rate,
      timeUnit: '1s',
      duration,
      preAllocatedVUs: Math.max(5, rate * 2),
      maxVUs: Math.max(20, rate * 5),
      tags: { role: 'producer' },
    },
    observer: {
      executor: 'constant-vus',
      exec: 'observe',
      vus: 1,
      duration,
      tags: { role: 'observer' },
    },
  },
  thresholds: {
    business_failures: ['rate<0.01'],
    dropped_iterations: ['count==0'],
  },
};

export function produce() {
  const response = http.post(`${baseURL}/shipments/mock/1`, null, {
    tags: { endpoint: '/shipments/mock/1' },
  });
  let ok = response.status === 200;
  try {
    ok = ok && response.json('code') === 1 && response.json('data.created') === 1;
  } catch (_) {
    ok = false;
  }
  businessFailures.add(!ok);
  check(response, { '成功创建一笔运单': () => ok });
}

export function observe() {
  const response = http.get(`${baseURL}/shipments?status=1`, {
    tags: { endpoint: '/shipments?status=1' },
  });
  let pending = -1;
  try {
    const data = response.json('data');
    pending = Array.isArray(data) ? data.length : -1;
  } catch (_) {
    pending = -1;
  }
  if (pending >= 0) {
    pendingShipments.add(pending, { input_rate: String(rate) });
  }
  sleep(1);
}
