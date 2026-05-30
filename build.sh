#!/bin/bash
set -e

echo "=== Building Vibe Time ==="

echo "[1/3] Building frontend..."
cd frontend && npm run build && cd ..

echo "[2/3] Building application..."
wails build

echo "[3/3] Copying frontend assets..."
mkdir -p build/bin/frontend
cp -rf frontend/dist build/bin/frontend/

echo ""
echo "Done!"

