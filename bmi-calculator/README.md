# BMI Calculator Microservices

A simple microservices-based BMI (Body Mass Index) calculator built in Go, designed for ArgoCD training and GitOps demonstrations.

## Architecture

The application consists of three microservices:

### 1. Gateway Service (Port 8080)
- **Purpose**: API Gateway that routes requests to appropriate services
- **Endpoints**:
  - `GET /health` - Health check for the gateway
  - `POST /api/calculate` - Calculate BMI with JSON payload
  - `GET /api/health` - Proxy to health service
  - `GET /api/bmi/*` - Proxy to BMI service

### 2. BMI Service (Port 8081)
- **Purpose**: Core BMI calculation logic and history tracking
- **Endpoints**:
  - `GET /health` - Health check
  - `POST /calculate` - Calculate BMI with JSON payload
  - `GET /bmi/{weight}/{height}` - Quick BMI calculation via URL parameters
  - `GET /history` - View calculation history

### 3. Health Service (Port 8082)
- **Purpose**: Comprehensive health monitoring and system information
- **Endpoints**:
  - `GET /health` - Basic health status
  - `GET /health/detailed` - Detailed system information
  - `GET /health/services` - Health status of all services
  - `GET /ready` - Readiness probe
  - `GET /live` - Liveness probe

## API Usage Examples

### Calculate BMI
```bash
curl -X POST http://localhost:8080/api/calculate \
  -H "Content-Type: application/json" \
  -d '{"weight": 70, "height": 1.75}'
```

Response:
```json
{
  "bmi": 22.86,
  "category": "Normal weight"
}
```

### Quick BMI Calculation
```bash
curl http://localhost:8080/api/bmi/70/1.75
```

### Health Check
```bash
curl http://localhost:8080/health
```

### Service Health Status
```bash
curl http://localhost:8080/api/health/services
```

## BMI Categories

- **Underweight**: BMI < 18.5
- **Normal weight**: 18.5 ≤ BMI < 25
- **Overweight**: 25 ≤ BMI < 30
- **Obese**: BMI ≥ 30

## Local Development

### Prerequisites
- Go 1.21+
- Docker (for containerization)

### Running Services

1. **Gateway Service**:
   ```bash
   cd gateway
   go run main.go
   ```

2. **BMI Service**:
   ```bash
   cd bmi-service
   go run main.go
   ```

3. **Health Service**:
   ```bash
   cd health-service
   go run main.go
   ```

### Building Docker Images

```bash
# Build all services
docker build -t bmi-calculator/gateway:latest ./gateway
docker build -t bmi-calculator/bmi-service:latest ./bmi-service
docker build -t bmi-calculator/health-service:latest ./health-service
```

## Kubernetes Deployment

The application includes Kubernetes manifests for deployment:

- `k8s/gateway/deployment.yaml` - Gateway service deployment
- `k8s/bmi-service/deployment.yaml` - BMI service deployment
- `k8s/health-service/deployment.yaml` - Health service deployment
- `k8s/kustomization.yaml` - Kustomize configuration
- `k8s/application.yaml` - ArgoCD Application manifest

### Deploy with kubectl

```bash
kubectl apply -k k8s/
```

### Deploy with ArgoCD

1. Update the `repoURL` in `k8s/application.yaml` to point to your repository
2. Apply the ArgoCD Application:
   ```bash
   kubectl apply -f k8s/application.yaml
   ```

## Features

- **Microservices Architecture**: Separate services for different concerns
- **Health Monitoring**: Comprehensive health checks and monitoring
- **Load Balancing**: Multiple replicas with service discovery
- **Container Ready**: Dockerfiles included for all services
- **GitOps Ready**: Kubernetes manifests for ArgoCD deployment
- **RESTful APIs**: Clean REST API design
- **History Tracking**: BMI calculation history in the BMI service
- **Environment Configuration**: Configurable via environment variables

## Environment Variables

### Gateway Service
- `PORT`: Service port (default: 8080)
- `BMI_SERVICE_URL`: BMI service URL (default: http://bmi-service:8081)
- `HEALTH_SERVICE_URL`: Health service URL (default: http://health-service:8082)

### BMI Service
- `PORT`: Service port (default: 8081)

### Health Service
- `PORT`: Service port (default: 8082)
- `ENVIRONMENT`: Environment name
- `NAMESPACE`: Kubernetes namespace
- `POD_NAME`: Pod name
- `POD_IP`: Pod IP address

## Perfect for ArgoCD Training

This application is ideal for ArgoCD training because it:

1. **Demonstrates Microservices**: Shows how to manage multiple services
2. **Includes Health Checks**: Proper liveness and readiness probes
3. **Uses Kustomize**: Shows how to use Kustomize for resource management
4. **GitOps Ready**: Complete GitOps workflow setup
5. **Simple but Complete**: Easy to understand but has real-world features
6. **Observable**: Multiple health endpoints for monitoring
7. **Scalable**: Supports multiple replicas and load balancing