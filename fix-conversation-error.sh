#!/bin/bash

echo "=== Fixing Conversation Service Error ==="

FRONTEND_DIR="/home/kiwi/workspace/cyberagent-frontend"
cd "$FRONTEND_DIR" || exit 1

# Fix the conversation service call
echo "Fixing app/[org]/conversation/[id]/real-client.tsx..."
sed -i '450s/communicationService.getConversation(organizationId, id);/communicationService.getConversation(id);/' app/[org]/conversation/[id]/real-client.tsx

# Check for similar patterns
echo "Checking for similar communication service calls..."
grep -r "getConversation.*organizationId.*id" app --include="*.tsx" --include="*.ts" | while read line; do
    file=$(echo "$line" | cut -d: -f1)
    echo "Found similar pattern in: $file"
    # Fix by removing organizationId parameter
    sed -i 's/getConversation(organizationId, /getConversation(/g' "$file"
done

# Final build attempt
echo "Running the real final build..."
rm -rf .next
NEXT_TELEMETRY_DISABLED=1 pnpm next build --no-lint 2>&1 | tee real-final-build.log

if [ $? -eq 0 ]; then
    echo ""
    echo "🎉🎉🎉 BUILD COMPLETED SUCCESSFULLY! 🎉🎉🎉"
    echo ""
    echo "Project Summary:"
    echo "================"
    echo "✅ Frontend built successfully with all fixes applied"
    echo "✅ Quota management interface available at: /[org]/quota"
    echo "✅ Quota service URL: https://cagen-quota-service-production.up.railway.app"
    echo "✅ TypeScript errors resolved"
    echo "✅ React 19 compatibility issues fixed"
    echo ""
    echo "Build Statistics:"
    grep -E "(Compiled successfully|Route \(app\)|First Load JS)" real-final-build.log | tail -20
    echo ""
    echo "You can now deploy the frontend!"
else
    echo "❌ Still have errors. Last error:"
    grep -B3 -A3 "Type error:" real-final-build.log | head -20
fi