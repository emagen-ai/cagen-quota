#!/bin/bash

# Generate a 32-byte base64 service key for Cagen Quota Service

echo "🔑 Generating Cagen Quota Service Key..."
echo ""

# Generate 32 random bytes and encode as base64
SERVICE_KEY=$(openssl rand -base64 32)

echo "📋 Generated Service Key:"
echo "CAGEN_QUOTA_SERVICE_SECRET_KEY=$SERVICE_KEY"
echo ""

echo "🔧 Configuration Instructions:"
echo ""
echo "1. For Railway deployment:"
echo "   - Go to your Railway project settings"
echo "   - Add environment variable: CAGEN_QUOTA_SERVICE_SECRET_KEY"
echo "   - Set value to: $SERVICE_KEY"
echo ""
echo "2. For local development:"
echo "   - Add to your .env file:"
echo "   - CAGEN_QUOTA_SERVICE_SECRET_KEY=$SERVICE_KEY"
echo ""
echo "3. For Docker deployment:"
echo "   - Add to docker-compose.yml environment section"
echo "   - Or pass as -e flag: -e CAGEN_QUOTA_SERVICE_SECRET_KEY='$SERVICE_KEY'"
echo ""

echo "⚠️  Security Notes:"
echo "   - Keep this key secret and secure"
echo "   - Use different keys for different environments"
echo "   - The key is exactly 32 bytes (44 characters in base64)"
echo "   - This key is used for encrypting communication with auth service"
echo ""

echo "✅ Key generation completed!"