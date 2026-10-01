cmake_minimum_required(VERSION 3.18)
project(reparos_bridge CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
set(CMAKE_CXX_VISIBILITY_PRESET hidden)
set(CMAKE_VISIBILITY_INLINES_HIDDEN ON)
if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
    set(CMAKE_BUILD_TYPE Release)
endif()

add_library(reparos_bridge SHARED bridge.cpp)
target_compile_definitions(reparos_bridge PRIVATE REPAROS_EXPORTS)
set_target_properties(reparos_bridge PROPERTIES OUTPUT_NAME "reparos_bridge" PREFIX "")

if(CT2_SOURCE_DIR AND SP_SOURCE_DIR)
    # Self-contained build: compile CTranslate2 and SentencePiece from source and
    # link them statically, so the resulting library has no extra runtime deps.
    #   cmake -S bridge -B build -DCT2_SOURCE_DIR=... -DSP_SOURCE_DIR=...
    set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)

    set(BUILD_CLI OFF CACHE BOOL "" FORCE)
    set(BUILD_TESTS OFF CACHE BOOL "" FORCE)
    set(WITH_MKL OFF CACHE BOOL "" FORCE)
    set(WITH_DNNL OFF CACHE BOOL "" FORCE)
    set(WITH_RUY ON CACHE BOOL "" FORCE)
    if(APPLE)
        set(WITH_ACCELERATE ON CACHE BOOL "" FORCE)
        set(OPENMP_RUNTIME NONE CACHE STRING "" FORCE)
    else()
        set(OPENMP_RUNTIME COMP CACHE STRING "" FORCE)
    endif()

    set(SPM_ENABLE_SHARED OFF CACHE BOOL "" FORCE)
    set(SPM_BUILD_TEST OFF CACHE BOOL "" FORCE)

    add_subdirectory(${CT2_SOURCE_DIR} ctranslate2 EXCLUDE_FROM_ALL)
    add_subdirectory(${SP_SOURCE_DIR} sentencepiece EXCLUDE_FROM_ALL)

    target_include_directories(reparos_bridge PRIVATE
        ${CT2_SOURCE_DIR}/include
        ${SP_SOURCE_DIR}/src)
    target_link_libraries(reparos_bridge PRIVATE ctranslate2 sentencepiece-static)
else()
    # Link against already-installed libraries: -DCT2_ROOT=... -DSP_ROOT=...
    if(CT2_ROOT)
        include_directories(${CT2_ROOT}/include)
        link_directories(${CT2_ROOT}/lib)
    endif()
    if(SP_ROOT)
        include_directories(${SP_ROOT}/include)
        link_directories(${SP_ROOT}/lib)
    endif()
    target_link_libraries(reparos_bridge PRIVATE ctranslate2 sentencepiece)
endif()

if(UNIX AND NOT APPLE)
    # Keep the C++ runtime inside the library and hide symbols from static deps.
    target_link_options(reparos_bridge PRIVATE -static-libstdc++ -static-libgcc -Wl,--exclude-libs,ALL)
endif()
