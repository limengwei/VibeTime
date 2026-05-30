#!/bin/bash
set -e

echo "=== Building Vibe Time ==="

echo "[1/3] Building application..."
wails build

echo "[2/3] Copying frontend assets..."
mkdir -p build/bin/frontend
cp -rf frontend/dist build/bin/frontend/

echo "[3/3] Creating NSIS installer..."
"/c/Program Files (x86)/NSIS/makensis" installer.nsi

echo ""
echo "Done! Installer: vibe-time-setup.exe"
ls -lh vibe-time-setup.exe
