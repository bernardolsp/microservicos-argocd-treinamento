import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

// Track which version we're hitting
const versionCounter = new Counter('version_hits');

// Configuration
const CANARY_URL = __ENV.CANARY_URL || 'http://localhost:8080';

export const options = {
  vus: 20,
  duration: '15m',
};

export default function () {
  const response = http.get(`${CANARY_URL}/`);
  
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'has version field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.version !== undefined;
      } catch (e) {
        return false;
      }
    },
  });
  
  if (success) {
    try {
      const body = JSON.parse(response.body);
      const version = body.version || 'unknown';
      const behavior = body.behavior || 'unknown';
      
      // Track version distribution
      versionCounter.add(1, { version: version, behavior: behavior });
      
      // Log periodically (every 100 requests)
      if (Math.random() < 0.01) {
        console.log(`Hit version=${version}, behavior=${behavior}`);
      }
    } catch (e) {
      // Ignore parse errors
    }
  }
  
  sleep(0.1);
}

export function handleSummary(data) {
  console.log('\n=== Canary Traffic Distribution ===');
  
  // Extract version distribution from metrics
  if (data.metrics.version_hits) {
    const tags = data.metrics.version_hits.tags;
    for (const [tag, count] of Object.entries(tags)) {
      console.log(`${tag}: ${count}`);
    }
  }
  
  return {
    stdout: JSON.stringify(data, null, 2),
  };
}
