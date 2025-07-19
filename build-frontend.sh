#!/bin/bash

echo "=== Building Frontend Project ==="
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

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    pnpm install
fi

# Install missing dependency for Progress component
echo "Installing @radix-ui/react-progress..."
pnpm add @radix-ui/react-progress

# Run build
echo "Running pnpm build..."
pnpm build

# Check build result
if [ $? -eq 0 ]; then
    echo "✅ Build completed successfully!"
else
    echo "❌ Build failed with errors"
    exit 1
fi