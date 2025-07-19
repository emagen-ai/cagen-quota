#!/bin/bash

echo "=== Building Frontend Project (Detailed) ==="
echo "Working directory: $(pwd)"

# Navigate to frontend directory
FRONTEND_DIR="/home/kiwi/workspace/cyberagent-frontend"

if [ ! -d "$FRONTEND_DIR" ]; then
    echo "Error: Frontend directory not found at $FRONTEND_DIR"
    exit 1
fi

echo "Changing to frontend directory: $FRONTEND_DIR"
cd "$FRONTEND_DIR" || exit 1

# Check if pnpm is installed
if ! command -v pnpm &> /dev/null; then
    echo "Error: pnpm is not installed"
    echo "Installing pnpm..."
    npm install -g pnpm
fi

# Check Node version
echo "Node version: $(node --version)"
echo "pnpm version: $(pnpm --version)"

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    pnpm install
fi

# Install missing dependency for Progress component
echo "Checking for @radix-ui/react-progress..."
if ! pnpm list @radix-ui/react-progress > /dev/null 2>&1; then
    echo "Installing @radix-ui/react-progress..."
    pnpm add @radix-ui/react-progress
fi

# Clean previous build
echo "Cleaning previous build..."
rm -rf .next

# Check TypeScript version
echo "TypeScript version:"
pnpm list typescript

# Run type check only
echo "Running type check..."
pnpm tsc --noEmit --skipLibCheck 2>&1 | head -50

# Try build with no lint
echo "Running build without lint..."
NEXT_TELEMETRY_DISABLED=1 pnpm next build --no-lint 2>&1 | tee build-output.log

# Check build result
if [ $? -eq 0 ]; then
    echo "✅ Build completed successfully!"
else
    echo "❌ Build failed with errors"
    echo "Last 50 lines of error output:"
    tail -50 build-output.log
    
    # Check for specific error patterns
    if grep -q "JSX.IntrinsicElements" build-output.log; then
        echo ""
        echo "⚠️  JSX type error detected. This might be a React version issue."
        echo "Current React version:"
        pnpm list react react-dom
    fi
fi