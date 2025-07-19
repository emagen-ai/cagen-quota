#!/bin/bash

echo "=== Fixing Action Types Issue ==="

FRONTEND_DIR="/home/kiwi/workspace/cyberagent-frontend"
cd "$FRONTEND_DIR" || exit 1

# Check the ActionDetail type definition
echo "Looking for ActionDetail type definition..."
grep -r "interface ActionDetail" --include="*.ts" --include="*.tsx" | head -10

# Check hub-service types
echo "Checking hub-service types..."
if [ -f "lib/services/hub-service.ts" ]; then
    grep -A20 "interface ActionDetail" lib/services/hub-service.ts || echo "ActionDetail not found in hub-service.ts"
fi

# Look for action types in general
echo "Looking for action type definitions..."
find . -path ./node_modules -prune -o -name "*.ts" -o -name "*.tsx" | xargs grep -l "creator_user_id.*mcp_server_ids" | head -10

# Fix the specific file by updating the setAction call
echo "Fixing app/[org]/actions/[id]/page.tsx..."
cat > /tmp/action-fix.patch << 'EOF'
--- a/app/[org]/actions/[id]/page.tsx
+++ b/app/[org]/actions/[id]/page.tsx
@@ -45,7 +45,12 @@
     setLoading(true);
     try {
       const data = await hubService.getAction(actionId);
-      setAction(data);
+      setAction({
+        ...data,
+        creator_user_id: data.creator_user_id || '',
+        mcp_server_ids: data.mcp_server_ids || []
+      } as ActionDetail);
     } catch (error) {
       console.error('Failed to load action:', error);
     } finally {
EOF

# Apply the patch
patch -p1 < /tmp/action-fix.patch || {
    echo "Patch failed, trying direct sed replacement..."
    sed -i '48s/setAction(data);/setAction({...data, creator_user_id: data.creator_user_id || "", mcp_server_ids: data.mcp_server_ids || []} as ActionDetail);/' "app/[org]/actions/[id]/page.tsx"
}

# Alternative fix - update the type definition if needed
echo "Checking if we need to update type definitions..."
if [ -f "lib/types/hub-types.ts" ]; then
    echo "Found hub-types.ts, checking content..."
    grep -A10 "interface Action" lib/types/hub-types.ts
fi

# Run build again
echo "Attempting build with type fixes..."
NEXT_TELEMETRY_DISABLED=1 pnpm next build --no-lint 2>&1 | tee build-types-fix.log

if [ $? -eq 0 ]; then
    echo "✅ Build succeeded with type fixes!"
else
    echo "❌ Build still failing. Showing type errors..."
    grep -A3 -B3 "Type error:" build-types-fix.log | head -30
fi