#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="${SCRIPT_DIR}/bmi-calculator"

DEFAULT_REGISTRY="docker"
DEFAULT_TAG="latest"
DEFAULT_DOCKER_USERNAME=""
DEFAULT_AWS_REGION="us-west-2"
DEFAULT_AWS_ACCOUNT_ID=""

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build and push BMI Calculator microservices to container registry.

OPTIONS:
    -r, --registry REGISTRY     Registry type (docker|ecr) [default: docker]
    -u, --username USERNAME     Docker Hub username (required for docker registry)
    -t, --tag TAG              Image tag [default: latest]
    -a, --account-id ACCOUNT_ID AWS Account ID (required for ecr registry)
    -R, --region REGION        AWS Region [default: us-west-2]
    -p, --push                 Push images to registry (default: build only)
    -h, --help                 Show this help message

EXAMPLES:
    # Build for Docker Hub
    $0 --registry docker --username myuser --tag v1.0.0 --push

    # Build for AWS ECR
    $0 --registry ecr --account-id 123456789012 --region us-east-1 --tag v1.0.0 --push

    # Build only (no push)
    $0 --registry docker --username myuser --tag v1.0.0

EOF
}

parse_args() {
    REGISTRY="$DEFAULT_REGISTRY"
    TAG="$DEFAULT_TAG"
    DOCKER_USERNAME="$DEFAULT_DOCKER_USERNAME"
    AWS_REGION="$DEFAULT_AWS_REGION"
    AWS_ACCOUNT_ID="$DEFAULT_AWS_ACCOUNT_ID"
    PUSH_IMAGES=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            -r|--registry)
                REGISTRY="$2"
                shift 2
                ;;
            -u|--username)
                DOCKER_USERNAME="$2"
                shift 2
                ;;
            -t|--tag)
                TAG="$2"
                shift 2
                ;;
            -a|--account-id)
                AWS_ACCOUNT_ID="$2"
                shift 2
                ;;
            -R|--region)
                AWS_REGION="$2"
                shift 2
                ;;
            -p|--push)
                PUSH_IMAGES=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

validate_args() {
    if [[ "$REGISTRY" != "docker" && "$REGISTRY" != "ecr" ]]; then
        echo "Error: Registry must be 'docker' or 'ecr'"
        exit 1
    fi

    if [[ "$REGISTRY" == "docker" && -z "$DOCKER_USERNAME" ]]; then
        echo "Error: Docker Hub username is required for docker registry"
        echo "Use: $0 --registry docker --username YOUR_USERNAME"
        exit 1
    fi

    if [[ "$REGISTRY" == "ecr" && -z "$AWS_ACCOUNT_ID" ]]; then
        echo "Error: AWS Account ID is required for ECR registry"
        echo "Use: $0 --registry ecr --account-id YOUR_ACCOUNT_ID"
        exit 1
    fi
}

check_prerequisites() {
    echo "Checking prerequisites..."

    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is not installed or not in PATH"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        echo "Error: Docker daemon is not running"
        exit 1
    fi

    if [[ "$REGISTRY" == "ecr" ]]; then
        if ! command -v aws &> /dev/null; then
            echo "Error: AWS CLI is not installed or not in PATH"
            exit 1
        fi
        
        if ! aws sts get-caller-identity &> /dev/null; then
            echo "Error: AWS CLI is not configured or credentials are invalid"
            echo "Run: aws configure"
            exit 1
        fi
    fi

    echo "✓ Prerequisites check passed"
}

setup_ecr() {
    echo "Setting up ECR registry..."
    
    ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
    
    echo "Logging into ECR..."
    if ! aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "$ECR_REGISTRY"; then
        echo "Error: ECR login failed"
        exit 1
    fi
    
    REPOSITORIES=(
        "bmi-calculator-gateway"
        "bmi-calculator-bmi-service"
        "bmi-calculator-health-service"
    )
    
    for repo in "${REPOSITORIES[@]}"; do
        echo "Creating ECR repository: $repo"
        aws ecr create-repository --repository-name "$repo" --region "$AWS_REGION" || true
    done
    
    echo "✓ ECR setup completed"
}

check_docker_login() {
    echo "Checking Docker Hub authentication..."
    
    if docker info 2>/dev/null | grep -q "Username.*$DOCKER_USERNAME"; then
        echo "✓ Already logged into Docker Hub as $DOCKER_USERNAME"
        return 0
    fi
    
    echo "Not logged into Docker Hub or different user detected"
    return 1
}

setup_docker() {
    echo "Setting up Docker Hub registry..."
    
    if ! check_docker_login; then
        echo "Please enter your Docker Hub password:"
        if ! docker login -u "$DOCKER_USERNAME"; then
            echo "Error: Docker login failed"
            exit 1
        fi
    fi
    
    echo "✓ Docker Hub setup completed"
}

build_image() {
    local service_name="$1"
    local dockerfile_path="$2"
    local image_name="$3"
    
    echo "Building $service_name image..."
    
    cd "$APP_DIR"
    docker build -t "${image_name}:${TAG}" -f "$dockerfile_path" .
    
    if [[ "$PUSH_IMAGES" == true ]]; then
        echo "Tagging $service_name for registry..."
        if [[ "$REGISTRY" == "docker" ]]; then
            FULL_IMAGE_NAME="${DOCKER_USERNAME}/${image_name}:${TAG}"
        else
            FULL_IMAGE_NAME="${ECR_REGISTRY}/${image_name}:${TAG}"
        fi
        
        docker tag "${image_name}:${TAG}" "$FULL_IMAGE_NAME"
        echo "Pushing $service_name..."
        docker push "$FULL_IMAGE_NAME"
        echo "✓ $service_name pushed to registry"
    else
        echo "✓ $service_name built locally"
    fi
}

build_images() {
    echo "Building Docker images..."
    
    SERVICES=(
        "gateway:gateway/Dockerfile:bmi-calculator-gateway"
        "bmi-service:bmi-service/Dockerfile:bmi-calculator-bmi-service"
        "health-service:health-service/Dockerfile:bmi-calculator-health-service"
    )
    
    for service in "${SERVICES[@]}"; do
        IFS=':' read -r name dockerfile image_name <<< "$service"
        build_image "$name" "$dockerfile" "$image_name"
    done
}

update_k8s_manifests() {
    echo "Updating Kubernetes manifests..."

    local image_prefix
    if [[ "$REGISTRY" == "docker" ]]; then
        image_prefix="${DOCKER_USERNAME}/"
    else
        image_prefix="${ECR_REGISTRY}/"
    fi

    cd "$APP_DIR/k8s"

    # Detect OS for sed compatibility
    local sed_inplace
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS requires no space between -i and extension
        sed_inplace="-i.bak"
    else
        # Linux sed requires space or empty string for -i
        sed_inplace="-i.bak"
    fi

    sed $sed_inplace "s|image: bmi-calculator-gateway:latest|image: ${image_prefix}bmi-calculator-gateway:${TAG}|g" gateway/deployment.yaml
    sed $sed_inplace "s|image: bmi-calculator-bmi-service:latest|image: ${image_prefix}bmi-calculator-bmi-service:${TAG}|g" bmi-service/deployment.yaml
    sed $sed_inplace "s|image: bmi-calculator-health-service:latest|image: ${image_prefix}bmi-calculator-health-service:${TAG}|g" health-service/deployment.yaml

    echo "✓ Kubernetes manifests updated"
    echo "  Backup files created with .bak extension"
}

show_summary() {
    echo ""
    echo "=== Build Summary ==="
    echo "Registry: $REGISTRY"
    echo "Tag: $TAG"
    echo "Push to registry: $PUSH_IMAGES"
    
    if [[ "$REGISTRY" == "docker" ]]; then
        echo "Docker Hub username: $DOCKER_USERNAME"
        echo "Images: ${DOCKER_USERNAME}/bmi-calculator-*:${TAG}"
    else
        echo "AWS Account ID: $AWS_ACCOUNT_ID"
        echo "AWS Region: $AWS_REGION"
        echo "ECR Registry: $ECR_REGISTRY"
        echo "Images: ${ECR_REGISTRY}/bmi-calculator-*:${TAG}"
    fi
    
    echo ""
    echo "Next steps:"
    echo "1. Deploy to Kubernetes: kubectl apply -k $APP_DIR/k8s"
    echo "2. Or use ArgoCD: kubectl apply -f $APP_DIR/k8s/application.yaml"
}

main() {
    parse_args "$@"
    validate_args
    check_prerequisites
    
    echo "Starting BMI Calculator build process..."
    echo "Registry: $REGISTRY, Tag: $TAG"
    
    if [[ "$REGISTRY" == "ecr" ]]; then
        setup_ecr
    else
        setup_docker
    fi
    
    build_images
    update_k8s_manifests
    show_summary
    
    echo ""
    echo "✓ Build process completed successfully!"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi