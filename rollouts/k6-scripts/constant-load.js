import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');

// Configuration - adjust these for your environment
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const RPS = parseInt(__ENV.RPS || '10');

export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: RPS,
      timeUnit: '1s',
      duration: '20m',
      preAllocatedVUs: 10,
      maxVUs: 50,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  const endpoints = ['/', '/health', '/api/data'];
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  const url = `${BASE_URL}${endpoint}`;
  
  const start = Date.now();
  const response = http.get(url);
  const duration = Date.now() - start;
  
  responseTime.add(duration);
  
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
  });
  
  errorRate.add(!success);
  
  // Minimal sleep for continuous load
  sleep(0.01);
}
