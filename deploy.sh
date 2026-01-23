#!/bin/bash

# Script to automate Docker build, microk8s import, and Kubernetes deployment update
# Usage: ./deploy.sh [version] [yaml-file]
# Example: ./deploy.sh 1.12 django.yaml

set -e  # Exit on error

# Configuration
IMAGE_NAME="mangabase-django"
NAMESPACE="mangafoxy"
DEPLOYMENT_NAME="django"
DEFAULT_VERSION="1.8"
DEFAULT_YAML_FILE="django.yaml"

# Get version from argument or use default
VERSION=${1:-$DEFAULT_VERSION}
YAML_FILE=${2:-$DEFAULT_YAML_FILE}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Django Deployment Automation ===${NC}"
echo -e "Image: ${IMAGE_NAME}:${VERSION}"
echo -e "YAML File: ${YAML_FILE}"
echo ""

# Step 1: Build Docker image
echo -e "${YELLOW}[1/5] Building Docker image...${NC}"
docker build -t ${IMAGE_NAME}:${VERSION} .
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Docker build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker image built successfully${NC}"
echo ""

# Step 2: Save and import to microk8s
echo -e "${YELLOW}[2/5] Importing image to microk8s...${NC}"
docker save ${IMAGE_NAME}:${VERSION} | sudo microk8s ctr image import -
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to import image to microk8s${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Image imported to microk8s successfully${NC}"
echo ""

# Step 3: Update YAML file with new image version
echo -e "${YELLOW}[3/5] Updating YAML file...${NC}"
if [ ! -f "$YAML_FILE" ]; then
    echo -e "${RED}Error: YAML file '$YAML_FILE' not found${NC}"
    exit 1
fi

# Create backup
cp "$YAML_FILE" "${YAML_FILE}.backup"
echo -e "${GREEN}✓ Backup created: ${YAML_FILE}.backup${NC}"

# Update image version in YAML file using sed
# This updates the image line: image: mangabase-django:OLD_VERSION
sed -i.tmp "s|image: ${IMAGE_NAME}:[0-9.]*|image: ${IMAGE_NAME}:${VERSION}|g" "$YAML_FILE"
rm -f "${YAML_FILE}.tmp"

echo -e "${GREEN}✓ YAML file updated with image ${IMAGE_NAME}:${VERSION}${NC}"
echo ""

# Step 4: Apply updated YAML to Kubernetes (microk8s)
echo -e "${YELLOW}[4/5] Applying updated deployment to Kubernetes...${NC}"
microk8s kubectl apply -f "$YAML_FILE" -n "$NAMESPACE"
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to apply YAML file${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Deployment updated successfully${NC}"
echo ""

# Step 5: Restart deployment
echo -e "${YELLOW}[5/5] Restarting deployment...${NC}"
microk8s kubectl rollout restart deployment/${DEPLOYMENT_NAME} -n "$NAMESPACE"
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to restart deployment${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Deployment restart initiated${NC}"
echo ""

# Wait for rollout to complete
echo -e "${YELLOW}Waiting for rollout to complete...${NC}"
microk8s kubectl rollout status deployment/${DEPLOYMENT_NAME} -n "$NAMESPACE" --timeout=5m
if [ $? -ne 0 ]; then
    echo -e "${YELLOW}Warning: Rollout status check timed out or failed${NC}"
    echo -e "${YELLOW}Check deployment status manually with: microk8s kubectl get pods -n ${NAMESPACE}${NC}"
else
    echo -e "${GREEN}✓ Rollout completed successfully${NC}"
fi

echo ""
echo -e "${GREEN}=== Deployment Complete ===${NC}"
echo -e "Image: ${IMAGE_NAME}:${VERSION}"
echo -e "Namespace: ${NAMESPACE}"
echo -e "Deployment: ${DEPLOYMENT_NAME}"
echo ""
echo -e "Check status with:"
echo -e "  microk8s kubectl get pods -n ${NAMESPACE}"
echo -e "  microk8s kubectl get deployment ${DEPLOYMENT_NAME} -n ${NAMESPACE}"
