cmake_minimum_required(VERSION 3.15)
project(reparos_bridge CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find or specify CTranslate2 and SentencePiece include/lib paths
# Can be provided via -DCT2_ROOT=... -DSP_ROOT=...
if(CT2_ROOT)
    include_directories(${CT2_ROOT}/include)
    link_directories(${CT2_ROOT}/lib)
endif()

if(SP_ROOT)
    include_directories(${SP_ROOT}/include)
    link_directories(${SP_ROOT}/lib)
endif()

add_library(reparos_bridge SHARED bridge.cpp)

target_compile_definitions(reparos_bridge PRIVATE REPAROS_EXPORTS)

target_link_libraries(reparos_bridge PRIVATE ctranslate2 sentencepiece)

set_target_properties(reparos_bridge PROPERTIES
    OUTPUT_NAME "reparos_bridge"
    PREFIX ""
)
