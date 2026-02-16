# Demo Applications for Argo Rollouts

This directory contains demo applications designed to test Argo Rollouts deployment strategies. The applications have configurable behaviors that simulate real-world scenarios including normal operation, errors, and high latency.

## Overview

```
rollouts/
├── app-src/                    # Application source code
│   ├── main.go                # Go application with Prometheus metrics
│   ├── go.mod                 # Go module definition
│   ├── Dockerfile             # Main Dockerfile
│   ├── v1/Dockerfile          # Version 1: Normal behavior
│   ├── v2/Dockerfile          # Version 2: Error-prone (50% errors)
│   └── v3/Dockerfile          # Version 3: Slow (high latency)
│
├── k8s/                       # Kubernetes manifests
│   ├── blue-green-rollout.yaml
│   ├── canary-rollout.yaml
│   ├── canary-with-metrics-rollout.yaml
│   ├── services.yaml
│   ├── servicemonitors.yaml
│   └── kustomization.yaml
│
├── k6-scripts/                # K6 load testing scripts
│   ├── load-test.js           # Standard load test
│   ├── constant-load.js       # Constant RPS load
│   ├── spike-test.js          # Spike test
│   ├── blue-green-test.js     # Blue/Green specific test
│   ├── canary-traffic-test.js # Canary traffic distribution test
│   └── run-k6.sh              # Helper script
│
└── traffic-generator/         # In-cluster traffic generation
    ├── configmap.yaml         # K6 scripts ConfigMap
    ├── deployment.yaml        # Continuous load Deployment
    ├── job-load-test.yaml     # One-time load test Job
    ├── cronjob.yaml           # Periodic load test CronJob
    └── kustomization.yaml
```

## Demo Application

### Behavior Modes

The demo application can operate in different modes controlled by the `BEHAVIOR` environment variable:

| Version | Behavior | Description | Use Case |
|---------|----------|-------------|----------|
| v1.0 | `normal` | Returns 200 OK, fast responses | Baseline, should pass all analysis |
| v2.0 | `error-prone` | 50% chance of 500 errors | Should fail analysis, trigger rollback |
| v3.0 | `slow` | 200-1000ms artificial delay | Should fail latency analysis |
| - | `chaotic` | Mix of slow and errors | Extreme failure scenario |

### Endpoints

- `GET /` - Root endpoint returning version info
- `GET /health` - Health check endpoint
- `GET /api/data` - Returns random data
- `GET /api/process` - Simulates processing (slower in `slow` mode)
- `GET /metrics` - Prometheus metrics

### Metrics Exposed

- `http_requests_total` - Counter with labels: method, endpoint, status
- `http_request_duration_seconds` - Histogram with labels: method, endpoint
- `app_version_info` - Gauge with version, behavior, hostname labels

## Building the Application

### Local Build

```bash
cd app-src/

# Build Go binary
go build -o demo-app .

# Run locally
VERSION=1.0 BEHAVIOR=normal ./demo-app
```

### Docker Build

```bash
cd app-src/

# Build version 1 (normal)
docker build -t demo-app:v1.0 -f v1/Dockerfile .

# Build version 2 (error-prone)
docker build -t demo-app:v2.0 -f v2/Dockerfile ..

# Build version 3 (slow)
docker build -t demo-app:v3.0 -f v3/Dockerfile ..

# Load into Kind cluster
kind load docker-image demo-app:v1.0 demo-app:v2.0 demo-app:v3.0
```

## Deployment

### Deploy All Demo Apps

```bash
# Deploy analysis templates first
kubectl apply -k ../../analysis-templates/

# Deploy demo applications
kubectl apply -k k8s/

# Deploy traffic generator
kubectl apply -k traffic-generator/
```

### Deploy Individual Apps

```bash
# Blue-Green only
kubectl apply -f k8s/services.yaml
kubectl apply -f k8s/servicemonitors.yaml
kubectl apply -f k8s/blue-green-rollout.yaml

# Canary with metrics only
kubectl apply -f k8s/services.yaml
kubectl apply -f k8s/servicemonitors.yaml
kubectl apply -f k8s/canary-with-metrics-rollout.yaml
```

## Testing Rollouts

### Scenario 1: Successful Blue-Green Deployment

```bash
# 1. Deploy v1 (normal)
kubectl apply -k k8s/

# 2. Start traffic generator
kubectl apply -k traffic-generator/

# 3. Watch rollout status
kubectl argo rollouts get rollout demo-app-blue-green --watch

# 4. Update to v2 (in a new terminal)
kubectl argo rollouts set image demo-app-blue-green \
  demo-app=demo-app:v2.0

# 5. Check preview service (should show errors)
kubectl port-forward svc/demo-app-blue-green-preview 8081:80
# In another terminal:
curl http://localhost:8081/health

# 6. Abort the rollout (v2 has errors!)
kubectl argo rollouts abort demo-app-blue-green

# 7. Try v3 (slow but works)
kubectl argo rollouts set image demo-app-blue-green \
  demo-app=demo-app:v3.0

# 8. Check preview (should be slow but return 200)
curl http://localhost:8081/health

# 9. Promote v3
kubectl argo rollouts promote demo-app-blue-green
```

### Scenario 2: Canary with Auto-Progression (Success)

```bash
# 1. Deploy v1
kubectl apply -k k8s/

# 2. Ensure traffic is flowing (important for analysis!)
kubectl apply -k traffic-generator/

# 3. Watch the rollout
kubectl argo rollouts get rollout demo-app-canary-metrics --watch

# 4. Update to v3 (slow but functional)
kubectl argo rollouts set image demo-app-canary-metrics \
  demo-app=demo-app:v3.0

# 5. Watch auto-progression
# The rollout should:
# - Go to 10% traffic
# - Run analysis (may fail if too slow)
# - Auto-promote or abort based on metrics
```

### Scenario 3: Canary with Auto-Progression (Failure)

```bash
# 1. Deploy v1
kubectl apply -k k8s/

# 2. Start traffic generator
kubectl apply -k traffic-generator/

# 3. Watch the rollout
kubectl argo rollouts get rollout demo-app-canary-metrics --watch

# 4. Update to v2 (error-prone - should fail analysis!)
kubectl argo rollouts set image demo-app-canary-metrics \
  demo-app=demo-app:v2.0

# 5. Watch analysis failure and auto-abort
# The rollout should:
# - Go to 10% traffic
# - Run analysis
# - Detect high error rate
# - Automatically abort and rollback
```

## Traffic Generation

### Local K6 (from your machine)

```bash
cd k6-scripts/

# Standard load test
./run-k6.sh load http://localhost:8080

# Constant load (10 RPS)
./run-k6.sh constant http://localhost:8080

# High constant load (50 RPS)
./run-k6.sh high-constant http://localhost:8080

# Spike test
./run-k6.sh spike http://localhost:8080

# Blue-Green test (tests both services)
./run-k6.sh blue-green http://active:80 http://preview:80

# Canary traffic distribution test
./run-k6.sh canary http://canary:80
```

### In-Cluster Traffic Generator

```bash
# Continuous load (Deployment)
kubectl apply -f traffic-generator/deployment.yaml

# One-time load test (Job)
kubectl apply -f traffic-generator/job-load-test.yaml

# Periodic load test (CronJob - every 15 minutes)
kubectl apply -f traffic-generator/cronjob.yaml

# Check traffic generator logs
kubectl logs -l app=traffic-generator -f
```

## K6 Test Scenarios

### 1. Load Test (`load-test.js`)

Ramps up traffic gradually:
- 1 min: 10 users
- 1 min: 25 users  
- 5 min: 50 users
- 10 min: 50 users (steady)
- 2 min: 10 users
- 1 min: 0 users

**Use for**: General load testing, validating performance under varying load

### 2. Constant Load (`constant-load.js`)

Maintains a constant request rate:
- Configurable RPS (default: 10)
- Duration: 20 minutes

**Use for**: Continuous traffic for canary analysis

### 3. Spike Test (`spike-test.js`)

Tests behavior under sudden load spikes:
- 30s: 5 users
- 30s: 100 users (spike)
- 2 min: 100 users
- 30s: 5 users
- 5 min: 10 users

**Use for**: Testing auto-scaling, circuit breakers

### 4. Blue-Green Test (`blue-green-test.js`)

Tests both active and preview services:
- Hits both services
- Logs version info
- Validates responses

**Use for**: Validating Blue-Green deployments

### 5. Canary Traffic Test (`canary-traffic-test.js`)

Monitors traffic distribution:
- Tracks which version receives traffic
- Logs version distribution
- Helps verify canary weights

**Use for**: Validating canary traffic splitting

## Monitoring

### Check Application Metrics

```bash
# Port-forward to Prometheus
kubectl port-forward svc/prometheus-server 9090:9090 -n monitoring

# Open http://localhost:9090 and query:
# Success rate:
# sum(rate(http_requests_total{service="demo-app-canary-metrics",status!~"5.."}[5m])) / sum(rate(http_requests_total{service="demo-app-canary-metrics"}[5m]))

# Error rate:
# sum(rate(http_requests_total{service="demo-app-canary-metrics",status=~"5.."}[5m])) / sum(rate(http_requests_total{service="demo-app-canary-metrics"}[5m]))

# P95 latency:
# histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{service="demo-app-canary-metrics"}[5m])) by (le))
```

### Watch Rollout Status

```bash
# Watch specific rollout
kubectl argo rollouts get rollout demo-app-canary-metrics --watch

# List all rollouts
kubectl argo rollouts list rollouts

# Get rollout details
kubectl describe rollout demo-app-canary-metrics

# Check analysis runs
kubectl get analysisruns
kubectl describe analysisrun <analysisrun-name>
```

## Troubleshooting

### Application pods not starting

```bash
# Check pod status
kubectl get pods -l app=demo-app-blue-green

# Check logs
kubectl logs -l app=demo-app-blue-green

# Check events
kubectl describe pods -l app=demo-app-blue-green
```

### No metrics in Prometheus

```bash
# Check ServiceMonitor
kubectl get servicemonitor

# Check if Prometheus sees the target
kubectl port-forward svc/prometheus-server 9090:9090 -n monitoring
# Open http://localhost:9090/targets

# Verify metrics endpoint
kubectl port-forward svc/demo-app-canary-metrics 8080:80
curl http://localhost:8080/metrics
```

### Analysis always fails

```bash
# Check if traffic is reaching the service
kubectl logs -l app=traffic-generator

# Test manually
kubectl port-forward svc/demo-app-canary-metrics 8080:80
for i in {1..20}; do curl -s http://localhost:8080/health; done

# Check Prometheus queries manually
```

### Rollout stuck

```bash
# Check rollout status
kubectl argo rollouts get rollout demo-app-canary-metrics

# Check for paused rollouts
kubectl get rollout demo-app-canary-metrics -o yaml | grep paused

# Abort if needed
kubectl argo rollouts abort demo-app-canary-metrics

# Retry
kubectl argo rollouts retry demo-app-canary-metrics
```

## Cleanup

```bash
# Remove all demo apps
kubectl delete -k k8s/

# Remove traffic generator
kubectl delete -k traffic-generator/

# Remove analysis templates
kubectl delete -k ../../analysis-templates/
```

## Integration with Course

### Day 4 - Argo Rollouts

These demo apps support the course curriculum:

1. **Blue-Green Deployment Theory**
   - Use `demo-app-blue-green` rollout
   - Deploy v1 → v2 (abort) → v3 (promote)
   - Show instant traffic switching

2. **Canary Deployment Theory**
   - Use `demo-app-canary` rollout
   - Show gradual traffic shifting
   - Manual promotion at each step

3. **Metrics-Based Auto-Progression**
   - Use `demo-app-canary-metrics` rollout
   - Deploy v2 (should auto-abort due to errors)
   - Deploy v3 (should auto-promote despite slowness)
   - Show Prometheus queries and analysis

4. **Pipeline Project**
   - Combine all components
   - Use ArgoCD Applications
   - Demonstrate GitOps workflow
