import http from 'k6/http';
import { check, sleep } from 'k6';

const baseUrl = (__ENV.BASE_URL || 'http://127.0.0.1:18080').replace(/\/$/, '');
const workOrderCode = __ENV.WORK_ORDER_CODE || 'OS-2026-0001';
const customerDocument = __ENV.CUSTOMER_DOCUMENT || '12345678901';
const peakVus = Number(__ENV.VUS || 200);
const warmupVus = Number(__ENV.WARMUP_VUS || Math.max(1, Math.ceil(peakVus / 10)));
const p95Ms = Number(__ENV.P95_MS || 500);
const latencyCheckMs = Number(__ENV.LATENCY_CHECK_MS || Math.max(p95Ms * 2, 1000));

export const options = {
  stages: [
    { duration: __ENV.WARMUP_DURATION || '30s', target: warmupVus },
    { duration: __ENV.RAMP_DURATION || '1m', target: peakVus },
    { duration: __ENV.PEAK_DURATION || '2m', target: peakVus },
    { duration: __ENV.COOLDOWN_DURATION || '30s', target: 0 },
  ],
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: ['rate<0.01'],
    http_req_duration: [`p(95)<${p95Ms}`],
  },
};

export default function () {
  const url = `${baseUrl}/public/work-orders/${encodeURIComponent(workOrderCode)}`
    + `?document=${encodeURIComponent(customerDocument)}`;
  const response = http.get(url, {
    tags: { name: 'GET /public/work-orders/:code' },
    timeout: __ENV.REQUEST_TIMEOUT || '10s',
  });

  let body = {};
  try {
    body = response.json();
  } catch (_) {
    // The checks below report invalid JSON without interrupting the scenario.
  }

  check(response, {
    'status is 200': (res) => res.status === 200,
    'response contains the requested work order code': () => body.code === workOrderCode,
    'response latency is within the per-request limit': (res) => res.timings.duration < latencyCheckMs,
  });

  sleep(Number(__ENV.SLEEP_SECONDS || 0.1));
}
