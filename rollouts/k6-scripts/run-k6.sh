#!/bin/bash

# K6 Traffic Generation Scripts
# Usage: ./run-k6.sh [load|constant|spike|blue-green|canary] [service-url]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_TYPE=${1:-load}
BASE_URL=${2:-http://canary.argocd.local:8080}

echo "Running k6 test: $TEST_TYPE"
echo "Target URL: $BASE_URL"
echo ""

case $TEST_TYPE in
  load)
    echo "Running standard load test..."
    k6 run --env BASE_URL=$BASE_URL "$SCRIPT_DIR/load-test.js"
    ;;
    
  constant)
    echo "Running constant load test (10 RPS)..."
    k6 run --env BASE_URL=$BASE_URL --env RPS=10 "$SCRIPT_DIR/constant-load.js"
    ;;
    
  high-constant)
    echo "Running high constant load test (50 RPS)..."
    k6 run --env BASE_URL=$BASE_URL --env RPS=50 "$SCRIPT_DIR/constant-load.js"
    ;;
    
  spike)
    echo "Running spike test..."
    k6 run --env BASE_URL=$BASE_URL "$SCRIPT_DIR/spike-test.js"
    ;;
    
  blue-green)
    ACTIVE_URL=${2:-http://localhost:8080}
    PREVIEW_URL=${3:-http://localhost:8081}
    echo "Running blue-green test..."
    echo "Active: $ACTIVE_URL"
    echo "Preview: $PREVIEW_URL"
    k6 run --env ACTIVE_URL=$ACTIVE_URL --env PREVIEW_URL=$PREVIEW_URL "$SCRIPT_DIR/blue-green-test.js"
    ;;
    
  canary)
    echo "Running canary traffic distribution test..."
    k6 run --env CANARY_URL=$BASE_URL "$SCRIPT_DIR/canary-traffic-test.js"
    ;;
    
  *)
    echo "Usage: $0 [load|constant|high-constant|spike|blue-green|canary] [url]"
    echo ""
    echo "Examples:"
    echo "  $0 load http://demo-app:80"
    echo "  $0 constant http://demo-app:80"
    echo "  $0 blue-green http://active:80 http://preview:80"
    echo "  $0 canary http://demo-app:80"
    exit 1
    ;;
esac
