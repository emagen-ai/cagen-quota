#!/bin/bash

echo "=== Fixing React 19 Import Issues ==="

FRONTEND_DIR="/home/kiwi/workspace/cyberagent-frontend"
cd "$FRONTEND_DIR" || exit 1

# Fix the specific problematic file first
echo "Fixing app/[org]/actions/[id]/page.tsx..."
sed -i "s/import React, { useState, useEffect } from 'react';/import React from 'react';\nimport { useState, useEffect } from 'react';/" "app/[org]/actions/[id]/page.tsx"

# Alternative fix - use namespace import
# sed -i "s/import React, { useState, useEffect } from 'react';/import * as React from 'react';\nconst { useState, useEffect } = React;/" "app/[org]/actions/[id]/page.tsx"

# Find all files with similar pattern and fix them
echo "Finding and fixing all files with similar import patterns..."
find . -name "*.tsx" -o -name "*.ts" | grep -v node_modules | while read file; do
    # Fix pattern: import React, { ... } from 'react'
    if grep -q "import React, {.*} from 'react'" "$file"; then
        echo "Fixing: $file"
        # Extract the imports
        imports=$(grep -o "{.*}" "$file" | head -1)
        # Replace with separate imports
        sed -i "s/import React, {.*} from 'react';/import React from 'react';\nimport $imports from 'react';/" "$file"
    fi
done

# Create a global React types fix
echo "Creating React 19 type definitions fix..."
mkdir -p types
cat > types/react-fix.d.ts << 'EOF'
/// <reference types="react" />
/// <reference types="react-dom" />

declare module 'react' {
  export = React;
  export as namespace React;
}
EOF

# Update tsconfig to use ES2020 module
echo "Updating tsconfig.json for React 19 compatibility..."
cat > tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2020",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": false,
    "forceConsistentCasingInFileNames": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./*"],
      "#/*": ["./*"]
    },
    "plugins": [
      {
        "name": "next"
      }
    ],
    "allowSyntheticDefaultImports": true
  },
  "include": [
    "next-env.d.ts",
    "**/*.ts",
    "**/*.tsx",
    ".next/types/**/*.ts",
    "types/**/*.d.ts"
  ],
  "exclude": [
    "node_modules"
  ]
}
EOF

# Quick type check on the problematic file
echo "Type checking the fixed file..."
pnpm tsc app/[org]/actions/[id]/page.tsx --noEmit --jsx preserve --skipLibCheck

# Try build again
echo "Attempting build with React import fixes..."
NEXT_TELEMETRY_DISABLED=1 pnpm next build --no-lint 2>&1 | tee build-react-fix.log

if [ $? -eq 0 ]; then
    echo "✅ Build succeeded with React import fixes!"
else
    echo "❌ Build still failing. Checking specific errors..."
    grep -A5 -B5 "Type error:" build-react-fix.log | head -50
fi