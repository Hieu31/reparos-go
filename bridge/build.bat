@echo off
echo Building reparos_bridge for Windows...
if not exist build mkdir build
cd build
cmake .. -A x64
cmake --build . --config Release
copy Release\reparos_bridge.dll ..\
echo Successfully built reparos_bridge.dll
cd ..
