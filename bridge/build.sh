#!/usr/bin/env bash
set -e

echo "Building reparos_bridge for Linux..."
mkdir -p build
cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
cmake --build . --config Release -j$(nproc)
cp libreparos_bridge.so ../
echo "Successfully built libreparos_bridge.so"
