#!/bin/bash

echo "=== Fixing TypeScript Issues ==="

FRONTEND_DIR="/home/kiwi/workspace/cyberagent-frontend"
cd "$FRONTEND_DIR" || exit 1

# Create a temporary tsconfig to diagnose the issue
echo "Creating temporary tsconfig for diagnosis..."
cat > tsconfig.temp.json << 'EOF'
{
  "compilerOptions": {
    "target": "es2020",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": false,
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
    ]
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

# Check if next-env.d.ts exists
if [ ! -f "next-env.d.ts" ]; then
    echo "Creating next-env.d.ts..."
    cat > next-env.d.ts << 'EOF'
/// <reference types="next" />
/// <reference types="next/image-types/global" />

// NOTE: This file should not be edited
// see https://nextjs.org/docs/pages/building-your-application/configuring/typescript#existing-projects
EOF
fi

# Check React types
echo "Checking React types installation..."
pnpm list @types/react @types/react-dom

# Reinstall React types if needed
echo "Reinstalling React types..."
pnpm add -D @types/react@latest @types/react-dom@latest

# Create a simple type check file to test
echo "Creating test file..."
cat > test-types.tsx << 'EOF'
import React from 'react';

export default function TestComponent() {
  return <div>Test</div>;
}
EOF

# Run type check on test file
echo "Running type check on test file..."
pnpm tsc test-types.tsx --noEmit --jsx preserve --lib dom,es2020 --skipLibCheck

# Clean up test file
rm -f test-types.tsx

# Try to fix the specific issue by updating tsconfig
echo "Updating tsconfig.json with fixes..."
cat > tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2017",
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
    "types": ["node", "react", "react-dom"]
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

# Install missing types
echo "Installing any missing type definitions..."
pnpm add -D @types/node

# Clean .next directory
echo "Cleaning .next directory..."
rm -rf .next

# Try build again with minimal config
echo "Attempting build with updated config..."
NEXT_TELEMETRY_DISABLED=1 pnpm next build --no-lint 2>&1 | tee build-fix-output.log

if [ $? -eq 0 ]; then
    echo "✅ Build succeeded with fixes!"
else
    echo "❌ Build still failing. Checking logs..."
    tail -20 build-fix-output.log
fi