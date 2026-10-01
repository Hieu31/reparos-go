cmake_minimum_required(VERSION 3.18)
project(reparos_bridge CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
if(MSVC)
    # Pass the standard as a flag: the CMake files of Abseil/SentencePiece use an old
    # cmake_minimum_required, so CMAKE_CXX_STANDARD is ignored by their feature checks.
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} /std:c++17 /Zc:__cplusplus /utf-8 /bigobj")
endif()

if(UNIX)
    # Everything is linked into one shared object, so all static deps must be PIC.
    # CTranslate2's own CMake unsets CMAKE_POSITION_INDEPENDENT_CODE for static
    # builds, so force the flag globally.
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fPIC")
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fPIC")
endif()
if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
    set(CMAKE_BUILD_TYPE Release)
endif()

add_library(reparos_bridge SHARED bridge.cpp)
target_compile_definitions(reparos_bridge PRIVATE REPAROS_EXPORTS)
set_target_properties(reparos_bridge PROPERTIES
    OUTPUT_NAME "reparos_bridge"
    PREFIX ""
    CXX_VISIBILITY_PRESET hidden
    VISIBILITY_INLINES_HIDDEN ON)

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

    # sentencepiece_processor.h includes "third_party/absl/...". SentencePiece links
    # that directory to its bundled abseil at configure time; copy it if that failed
    # (e.g. symlinks unavailable on Windows).
    if(NOT EXISTS ${SP_SOURCE_DIR}/third_party/absl)
        file(COPY ${SP_SOURCE_DIR}/third_party/abseil-cpp/absl DESTINATION ${SP_SOURCE_DIR}/third_party)
    endif()

    target_include_directories(reparos_bridge PRIVATE
        ${CT2_SOURCE_DIR}/include
        ${SP_SOURCE_DIR}
        ${SP_SOURCE_DIR}/src
        ${SP_SOURCE_DIR}/third_party/abseil-cpp)
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
