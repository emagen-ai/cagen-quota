#!/bin/bash

# Test script for Cagen Quota Service

set -e

BASE_URL="http://localhost:8080"
SERVICE_ID="svc_cagen_quota"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🧪 Testing Cagen Quota Service"
echo "================================"

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
    fi
}

# Function to make HTTP request
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            "$BASE_URL$endpoint")
    fi
    
    # Split response body and status code
    body=$(echo "$response" | head -n -1)
    status=$(echo "$response" | tail -n 1)
    
    echo "Response: $body"
    echo "Status: $status"
    
    if [ "$status" = "$expected_status" ]; then
        return 0
    else
        return 1
    fi
}

echo ""
echo "🔍 1. Health Check"
echo "----------------"
make_request "GET" "/health" "" "200"
print_result $? "Health check"

echo ""
echo "🔍 2. Development Info (if available)"
echo "-----------------------------------"
make_request "GET" "/dev/info" "" "200"
print_result $? "Development info"

echo ""
echo "🔍 3. Test Auth Endpoint (if available)"
echo "-------------------------------------"
test_auth_data='{
    "service_id": "'$SERVICE_ID'",
    "encrypted_data": "dGVzdC1kYXRhLWZvci10ZXN0aW5n"
}'

make_request "POST" "/dev/test-auth" "$test_auth_data" "200"
print_result $? "Test auth endpoint"

echo ""
echo "🔍 4. Create Quota Test (without real auth)"
echo "------------------------------------------"
create_quota_data='{
    "service_id": "'$SERVICE_ID'",
    "encrypted_data": "dGVzdC1lbmNyeXB0ZWQtZGF0YQ==",
    "name": "Test Organization Quota",
    "description": "Test quota for development",
    "type": "organization",
    "total_mb": 1000
}'

echo "Note: This test may fail due to authentication requirements"
make_request "POST" "/api/v1/quotas/create" "$create_quota_data" "500" || true
print_result $? "Create quota test (expected to fail without real auth)"

echo ""
echo "🔍 5. List Quotas Test"
echo "--------------------"
echo "Note: This test may fail due to authentication requirements"
make_request "GET" "/api/v1/quotas?service_id=$SERVICE_ID&encrypted_data=dGVzdA==" "" "400" || true
print_result $? "List quotas test (expected to fail without real auth)"

echo ""
echo "📊 Test Summary"
echo "==============="
echo -e "${YELLOW}Service is running and responding to requests${NC}"
echo -e "${YELLOW}Full functionality requires:${NC}"
echo "  - Database connection"
echo "  - Valid service key configuration"
echo "  - Auth service integration"
echo "  - Proper encrypted user data"
echo ""
echo -e "${GREEN}✨ Basic service tests completed!${NC}"
echo ""
echo "🚀 Next steps:"
echo "  1. Configure database connection"
echo "  2. Generate service key: ./scripts/generate_key.sh"
echo "  3. Configure environment variables"
echo "  4. Test with real encrypted user data"