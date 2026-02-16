import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    // Start minimal
    { duration: '30s', target: 5 },
    // Spike to high load
    { duration: '30s', target: 100 },
    // Maintain spike
    { duration: '2m', target: 100 },
    // Quick recovery
    { duration: '30s', target: 5 },
    // Back to normal
    { duration: '5m', target: 10 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.2'],
  },
};

export default function () {
  const url = `${BASE_URL}/api/process`;
  
  const start = Date.now();
  const response = http.get(url);
  const duration = Date.now() - start;
  
  responseTime.add(duration);
  
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 2000ms': (r) => duration < 2000,
  });
  
  errorRate.add(!success);
  
  sleep(0.05);
}
