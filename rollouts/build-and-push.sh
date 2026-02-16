#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_NAME="version-app"

DEFAULT_REGISTRY="docker"
DEFAULT_TAG="latest"
DEFAULT_DOCKER_USERNAME=""

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build and push Argo Rollouts demo app images to container registry.
Builds multiple versions: latest, v1, v2, v3

OPTIONS:
    -r, --registry REGISTRY     Registry type (docker) [default: docker]
    -u, --username USERNAME     Docker Hub username (required)
    -t, --tag TAG              Image tag [default: latest]
    -p, --push                 Push images to registry (default: build only)
    -a, --all-versions         Build all versions (v1, v2, v3) [default: latest only]
    -h, --help                 Show this help message

EXAMPLES:
    # Build latest only
    $0 --username myuser --tag v1.0.0

    # Build and push latest only
    $0 --username myuser --tag v1.0.0 --push

    # Build all versions
    $0 --username myuser --tag v1.0.0 --all-versions --push

EOF
}

parse_args() {
    REGISTRY="$DEFAULT_REGISTRY"
    TAG="$DEFAULT_TAG"
    DOCKER_USERNAME="$DEFAULT_DOCKER_USERNAME"
    PUSH_IMAGES=false
    BUILD_ALL=false

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
            -p|--push)
                PUSH_IMAGES=true
                shift
                ;;
            -a|--all-versions)
                BUILD_ALL=true
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
    if [[ -z "$DOCKER_USERNAME" ]]; then
        echo "Error: Docker Hub username is required"
        echo "Use: $0 --username YOUR_USERNAME"
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

    echo "✓ Prerequisites check passed"
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

build_version() {
    local version="$1"
    local dockerfile_path="$2"
    local image_tag="$3"
    
    echo ""
    echo "=== Building version: $version ==="
    
    local full_image_name="${DOCKER_USERNAME}/${APP_NAME}:${image_tag}"
    
    echo "Building image: $full_image_name"
    docker build -t "$full_image_name" -f "$dockerfile_path" "$SCRIPT_DIR/app-src"
    
    if [[ "$PUSH_IMAGES" == true ]]; then
        echo "Pushing to registry..."
        docker push "$full_image_name"
        echo "✓ Pushed: $full_image_name"
    else
        echo "✓ Built locally: $full_image_name"
    fi
}

build_images() {
    echo "Building Docker images..."
    
    # Build latest (main Dockerfile)
    build_version "latest" "$SCRIPT_DIR/app-src/Dockerfile" "$TAG"
    
    # Build specific versions if requested
    if [[ "$BUILD_ALL" == true ]]; then
        build_version "v1 (normal)" "$SCRIPT_DIR/app-src/v1/Dockerfile" "${TAG}-v1"
        build_version "v2 (error-prone)" "$SCRIPT_DIR/app-src/v2/Dockerfile" "${TAG}-v2"
        build_version "v3 (slow)" "$SCRIPT_DIR/app-src/v3/Dockerfile" "${TAG}-v3"
    fi
}

show_summary() {
    echo ""
    echo "=========================================="
    echo "        BUILD SUMMARY"
    echo "=========================================="
    echo ""
    echo "Docker Hub username: $DOCKER_USERNAME"
    echo "Image name: $APP_NAME"
    echo "Tag: $TAG"
    echo "Push to registry: $PUSH_IMAGES"
    echo "Build all versions: $BUILD_ALL"
    echo ""
    echo "Images:"
    echo "  - ${DOCKER_USERNAME}/${APP_NAME}:${TAG}"
    
    if [[ "$BUILD_ALL" == true ]]; then
        echo "  - ${DOCKER_USERNAME}/${APP_NAME}:${TAG}-v1 (normal)"
        echo "  - ${DOCKER_USERNAME}/${APP_NAME}:${TAG}-v2 (error-prone)"
        echo "  - ${DOCKER_USERNAME}/${APP_NAME}:${TAG}-v3 (slow)"
    fi
    
    echo ""
    echo "=========================================="
    echo "Next steps:"
    echo ""
    echo "1. Update rollout manifests with new image:"
    echo "   Image: ${DOCKER_USERNAME}/${APP_NAME}:${TAG}"
    echo ""
    echo "2. Deploy to Kubernetes:"
    echo "   kubectl apply -f k8s/services.yaml"
    echo "   kubectl apply -f k8s/canary-rollout.yaml"
    echo ""
    echo "3. Or use kubectl argo rollouts plugin:"
    echo "   kubectl argo rollouts get rollout demo-rollout"
    echo "   kubectl argo rollouts promote demo-rollout"
    echo ""
    echo "=========================================="
}

main() {
    parse_args "$@"
    validate_args
    check_prerequisites
    
    echo "=========================================="
    echo "Argo Rollouts Demo App Build"
    echo "=========================================="
    echo ""
    echo "Username: $DOCKER_USERNAME"
    echo "Tag: $TAG"
    echo "Push: $PUSH_IMAGES"
    echo "Build all versions: $BUILD_ALL"
    echo ""
    
    setup_docker
    build_images
    show_summary
    
    echo ""
    echo "✓ Build process completed successfully!"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
