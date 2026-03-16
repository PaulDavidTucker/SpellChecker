#!/bin/bash

# Check if port 8080 is already in use
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "ERROR: Port 8080 is already in use!"
    echo "Please stop the process using port 8080 or change the port mapping in docker-compose.yaml"
    echo ""
    echo "To find the process: lsof -i :8080"
    echo "To kill it: kill -9 \$(lsof -t -i :8080)"
    exit 1
fi

# Check if port 3000 is already in use
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "ERROR: Port 3000 is already in use!"
    echo "Please stop the process using port 3000 or change the port mapping in docker-compose.yaml"
    echo ""
    echo "To find the process: lsof -i :3000"
    echo "To kill it: kill -9 \$(lsof -t -i :3000)"
    exit 1
fi

echo "Ports 8080 and 3000 are available"
echo "Starting SpellChecker services..."
echo ""

# Start services
docker compose up --build
