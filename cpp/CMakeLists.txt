cmake_minimum_required(VERSION 3.20)
project(raidzero VERSION 0.2.1 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# ── options ────────────────────────────────────────────────────────────────
option(RAIDZERO_INSTALL "Install cmake targets and headers" ON)
option(RAIDZERO_EXAMPLES "Build the bundled examples" OFF)
option(RAIDZERO_FETCH_DEPS "Fetch missing C++ dependencies with FetchContent" ON)

find_package(nlohmann_json CONFIG QUIET)
find_package(cpr CONFIG QUIET)

if(NOT nlohmann_json_FOUND OR NOT cpr_FOUND)
    if(NOT RAIDZERO_FETCH_DEPS)
        message(FATAL_ERROR "Missing nlohmann_json or cpr. Install them or enable RAIDZERO_FETCH_DEPS.")
    endif()

    # fetch only dependencies that are not already available from the system.
    include(FetchContent)

    if(NOT nlohmann_json_FOUND)
        # fetched json needs an export set when we install the sdk.
        set(JSON_Install ${RAIDZERO_INSTALL} CACHE BOOL "" FORCE)
        FetchContent_Declare(
            nlohmann_json
            GIT_REPOSITORY https://github.com/nlohmann/json.git
            GIT_TAG        v3.11.3
            GIT_SHALLOW    TRUE
        )
        FetchContent_MakeAvailable(nlohmann_json)
    endif()

    if(NOT cpr_FOUND)
        FetchContent_Declare(
            cpr
            GIT_REPOSITORY https://github.com/libcpr/cpr.git
            GIT_TAG        1.10.5
            GIT_SHALLOW    TRUE
        )

        # only build what we need from cpr
        set(CPR_BUILD_TESTS     OFF CACHE BOOL "" FORCE)
        set(CPR_BUILD_EXAMPLES  OFF CACHE BOOL "" FORCE)
        set(CPR_USE_SYSTEM_CURL ON  CACHE BOOL "" FORCE)
        FetchContent_MakeAvailable(cpr)
    endif()
endif()

# ── interface library ──────────────────────────────────────────────────────
# header-only: INTERFACE target, no compiled translation units
add_library(raidzero INTERFACE)
add_library(raidzero::raidzero ALIAS raidzero)

target_include_directories(raidzero
    INTERFACE
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
)

target_link_libraries(raidzero
    INTERFACE
        nlohmann_json::nlohmann_json
        cpr::cpr
)

target_compile_features(raidzero INTERFACE cxx_std_17)

# ── optional examples ──────────────────────────────────────────────────────
if(RAIDZERO_EXAMPLES)
    add_subdirectory(examples)
endif()

# ── install rules ──────────────────────────────────────────────────────────
if(RAIDZERO_INSTALL)
    include(GNUInstallDirs)
    include(CMakePackageConfigHelpers)

    install(
        TARGETS raidzero
        EXPORT  raidzeroTargets
    )

    install(
        DIRECTORY include/
        DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}
    )

    install(
        EXPORT      raidzeroTargets
        FILE        raidzeroTargets.cmake
        NAMESPACE   raidzero::
        DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/raidzero
    )

    write_basic_package_version_file(
        "${CMAKE_CURRENT_BINARY_DIR}/raidzeroConfigVersion.cmake"
        VERSION       ${PROJECT_VERSION}
        COMPATIBILITY SameMajorVersion
    )

    configure_package_config_file(
        "${CMAKE_CURRENT_SOURCE_DIR}/cmake/raidzeroConfig.cmake.in"
        "${CMAKE_CURRENT_BINARY_DIR}/raidzeroConfig.cmake"
        INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/raidzero
    )

    install(
        FILES
            "${CMAKE_CURRENT_BINARY_DIR}/raidzeroConfig.cmake"
            "${CMAKE_CURRENT_BINARY_DIR}/raidzeroConfigVersion.cmake"
        DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/raidzero
    )
endif()
