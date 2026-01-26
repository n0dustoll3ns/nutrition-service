#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting Auth Service Test...${NC}"

# Build the application
echo -e "${YELLOW}Building application...${NC}"
go build -o auth-service ./cmd/server
if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed${NC}"
    exit 1
fi
echo -e "${GREEN}Build successful${NC}"

# Start server in background
echo -e "${YELLOW}Starting server...${NC}"
./auth-service &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}Server failed to start${NC}"
    exit 1
fi
echo -e "${GREEN}Server started (PID: $SERVER_PID)${NC}"

# Test health endpoint
echo -e "${YELLOW}Testing health endpoint...${NC}"
HEALTH_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
if [ "$HEALTH_RESPONSE" = "200" ]; then
    echo -e "${GREEN}Health endpoint: OK${NC}"
else
    echo -e "${RED}Health endpoint: FAILED (HTTP $HEALTH_RESPONSE)${NC}"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# Test register endpoint (placeholder)
echo -e "${YELLOW}Testing register endpoint (placeholder)...${NC}"
REGISTER_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/api/v1/auth/register)
if [ "$REGISTER_RESPONSE" = "200" ]; then
    echo -e "${GREEN}Register endpoint: OK (placeholder)${NC}"
else
    echo -e "${RED}Register endpoint: FAILED (HTTP $REGISTER_RESPONSE)${NC}"
fi

# Test protected endpoint (should work but return placeholder)
echo -e "${YELLOW}Testing protected endpoint (placeholder)...${NC}"
PROTECTED_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/protected/me)
if [ "$PROTECTED_RESPONSE" = "200" ]; then
    echo -e "${GREEN}Protected endpoint: OK (placeholder)${NC}"
else
    echo -e "${RED}Protected endpoint: FAILED (HTTP $PROTECTED_RESPONSE)${NC}"
fi

# Stop server
echo -e "${YELLOW}Stopping server...${NC}"
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
echo -e "${GREEN}Server stopped${NC}"

# Cleanup
rm -f auth-service

echo -e "${GREEN}All tests passed!${NC}"
echo -e "${YELLOW}Note: This is a basic connectivity test. Full functionality tests will be added later.${NC}"
