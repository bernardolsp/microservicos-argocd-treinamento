import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const requests = new Counter('requests');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TARGET_SERVICE = __ENV.TARGET_SERVICE || 'demo-app';

export const options = {
  stages: [
    // Ramp up
    { duration: '1m', target: 10 },
    { duration: '1m', target: 25 },
    // Steady state
    { duration: '5m', target: 50 },
    { duration: '10m', target: 50 },
    // Ramp down
    { duration: '2m', target: 10 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
    http_req_failed: ['rate<0.1'],    // Error rate under 10%
    errors: ['rate<0.1'],
  },
};

export default function () {
  const endpoints = [
    { url: '/', name: 'root' },
    { url: '/health', name: 'health' },
    { url: '/api/data', name: 'data' },
    { url: '/api/process', name: 'process' },
  ];

  // Randomly select an endpoint
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  const url = `${BASE_URL}${endpoint.url}`;
  
  const start = Date.now();
  const response = http.get(url);
  const duration = Date.now() - start;
  
  responseTime.add(duration);
  requests.add(1);
  
  const success = check(response, {
    [`${endpoint.name} status is 200`]: (r) => r.status === 200,
    [`${endpoint.name} response time < 1000ms`]: (r) => duration < 1000,
  });
  
  errorRate.add(!success);
  
  // Small sleep to prevent overwhelming the service
  sleep(Math.random() * 0.5 + 0.1);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

// Helper function for text summary
function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  const colors = {
    reset: enableColors ? '\x1b[0m' : '',
    red: enableColors ? '\x1b[31m' : '',
    green: enableColors ? '\x1b[32m' : '',
    yellow: enableColors ? '\x1b[33m' : '',
  };
  
  let summary = '';
  summary += `${indent}╔════════════════════════════════════════════════════════╗\n`;
  summary += `${indent}║           K6 Load Test Summary                         ║\n`;
  summary += `${indent}╚════════════════════════════════════════════════════════╝\n\n`;
  
  summary += `${indent}Target: ${BASE_URL}\n`;
  summary += `${indent}Service: ${TARGET_SERVICE}\n\n`;
  
  // HTTP metrics
  const httpReqs = data.metrics.http_reqs;
  const httpReqFailed = data.metrics.http_req_failed;
  const httpReqDuration = data.metrics.http_req_duration;
  
  summary += `${indent}Requests: ${httpReqs.values.count}\n`;
  summary += `${indent}Failed: ${httpReqFailed.values.rate * 100}%\n`;
  summary += `${indent}Duration (avg): ${httpReqDuration.values.avg}ms\n`;
  summary += `${indent}Duration (p95): ${httpReqDuration.values['p(95)']}ms\n`;
  summary += `${indent}Duration (p99): ${httpReqDuration.values['p(99)']}ms\n\n`;
  
  // Thresholds
  summary += `${indent}Thresholds:\n`;
  for (const [name, threshold] of Object.entries(data.thresholds)) {
    const status = threshold.ok ? colors.green + '✓' : colors.red + '✗';
    summary += `${indent}  ${status} ${name}${colors.reset}\n`;
  }
  
  return summary;
}
