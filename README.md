# Build and Deploy Scripts

This directory contains scripts for building and deploying the BMI Calculator microservices.

## build-and-push.sh

A comprehensive script to build and push Docker images to either Docker Hub or AWS ECR, and automatically update Kubernetes manifests with the correct image tags.

### Prerequisites

**For Docker Hub:**
- Docker installed and running
- Docker Hub account

**For AWS ECR:**
- Docker installed and running
- AWS CLI installed and configured
- AWS account with ECR permissions

### Usage

#### Docker Hub Registry

```bash
# Build and push to Docker Hub
./build-and-push.sh --registry docker --username YOUR_DOCKER_USERNAME --tag v1.0.0 --push

# Build only (no push)
./build-and-push.sh --registry docker --username YOUR_DOCKER_USERNAME --tag v1.0.0
```

#### AWS ECR Registry

```bash
# Build and push to ECR
./build-and-push.sh --registry ecr --account-id 123456789012 --region us-west-2 --tag v1.0.0 --push

# Build only (no push)
./build-and-push.sh --registry ecr --account-id 123456789012 --region us-west-2 --tag v1.0.0
```

### Options

| Option | Description |
|--------|-------------|
| `-r, --registry` | Registry type: `docker` or `ecr` (default: docker) |
| `-u, --username` | Docker Hub username (required for docker registry) |
| `-t, --tag` | Image tag (default: latest) |
| `-a, --account-id` | AWS Account ID (required for ecr registry) |
| `-R, --region` | AWS Region (default: us-west-2) |
| `-p, --push` | Push images to registry (default: build only) |
| `-h, --help` | Show help message |

### What the script does:

1. **Validation**: Checks prerequisites and validates arguments
2. **Registry Setup**: 
   - For Docker Hub: Prompts for password and logs in
   - For ECR: Creates repositories if they don't exist and logs in
3. **Build**: Builds all three service images (gateway, bmi-service, health-service)
4. **Tag & Push**: Tags images with registry prefix and pushes if requested
5. **Update Manifests**: Automatically updates Kubernetes deployment files with correct image paths
6. **Summary**: Shows what was built and next steps

### Examples

#### Quick start with Docker Hub:
```bash
cd apps
./build-and-push.sh --registry docker --username mydockeruser --tag v1.0.0 --push
```

This will:
- Build all three images
- Tag them as `mydockeruser/bmi-calculator-gateway:v1.0.0`, etc.
- Push to Docker Hub
- Update Kubernetes manifests to use the new images

#### Quick start with AWS ECR:
```bash
cd apps
./build-and-push.sh --registry ecr --account-id 123456789012 --region us-east-1 --tag v1.0.0 --push
```

This will:
- Build all three images
- Tag them as `123456789012.dkr.ecr.us-east-1.amazonaws.com/bmi-calculator-gateway:v1.0.0`, etc.
- Push to ECR
- Update Kubernetes manifests to use the new images

### After building

Deploy to Kubernetes:
```bash
kubectl apply -k ../bmi-calculator/k8s
```

Or use ArgoCD:
```bash
kubectl apply -f ../bmi-calculator/k8s/application.yaml
```

### Image Naming Convention

**Docker Hub:**
- `username/bmi-calculator-gateway:tag`
- `username/bmi-calculator-bmi-service:tag`
- `username/bmi-calculator-health-service:tag`

**AWS ECR:**
- `account-id.dkr.ecr.region.amazonaws.com/bmi-calculator-gateway:tag`
- `account-id.dkr.ecr.region.amazonaws.com/bmi-calculator-bmi-service:tag`
- `account-id.dkr.ecr.region.amazonaws.com/bmi-calculator-health-service:tag`

### Backup Files

The script creates `.bak` backup files of the original Kubernetes manifests before updating them. You can restore them if needed:

```bash
cd ../bmi-calculator/k8s
mv gateway/deployment.yaml.bak gateway/deployment.yaml
mv bmi-service/deployment.yaml.bak bmi-service/deployment.yaml
mv health-service/deployment.yaml.bak health-service/deployment.yaml
```