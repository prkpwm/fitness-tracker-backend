#!/bin/bash

echo "Building and deploying Fitness Tracker Backend..."

# Build Docker image
docker build -t fitness-backend .

# Stop existing container if running
docker stop fitness-backend 2>/dev/null || true
docker rm fitness-backend 2>/dev/null || true

# Run new container
docker run -d \
  --name fitness-backend \
  -p 8080:8080 \
  --restart unless-stopped \
  fitness-backend

echo "Backend deployed successfully on http://localhost:8080"
echo "API endpoints:"
echo "  GET  /api/fitness"
echo "  POST /api/fitness"
echo "  GET  /api/fitness/{date}"