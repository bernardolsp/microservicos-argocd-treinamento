import http from 'k6/http';
import { check, sleep } from 'k6';

// Configuration
const ACTIVE_URL = __ENV.ACTIVE_URL || 'http://localhost:8080';
const PREVIEW_URL = __ENV.PREVIEW_URL || 'http://localhost:8081';

export const options = {
  vus: 10,
  duration: '10m',
};

export default function () {
  // Test both active and preview services
  const services = [
    { name: 'active', url: ACTIVE_URL },
    { name: 'preview', url: PREVIEW_URL },
  ];
  
  for (const service of services) {
    const response = http.get(`${service.url}/`);
    
    check(response, {
      [`${service.name} status is 200`]: (r) => r.status === 200,
      [`${service.name} has version`]: (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.version !== undefined;
        } catch (e) {
          return false;
        }
      },
    });
    
    // Log version info
    try {
      const body = JSON.parse(response.body);
      console.log(`${service.name}: version=${body.version}, behavior=${body.behavior}`);
    } catch (e) {
      console.log(`${service.name}: error parsing response`);
    }
  }
  
  sleep(1);
}
