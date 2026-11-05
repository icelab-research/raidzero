"""
Conan 2 recipe for sf_voice — header-only C++17 helper for the sf-voice media API.

This is provided for convenience if you already use Conan. It is not published
to any remote registry. Use it locally or via CMake FetchContent instead.

Usage (local):
    conan install . --output-folder=build --build=missing
    cmake -B build -DCMAKE_TOOLCHAIN_FILE=build/conan_toolchain.cmake
    cmake --build build
"""

from conan import ConanFile
from conan.tools.cmake import CMake, CMakeToolchain, cmake_layout
from conan.tools.files import copy
import os


class SfVoiceMediaConan(ConanFile):
    name        = "sf_voice"
    version     = "0.1.1"
    license     = "MIT"
    description = "Header-only C++17 helper for the sf-voice media API"
    url         = "https://github.com/icelab-research/raidzero"
    homepage    = "https://sf-voice.com"
    topics      = ("media", "video", "audio", "http", "header-only")

    # header-only: no compiler/os/arch settings needed
    package_type = "header-library"
    settings     = "os", "compiler", "build_type", "arch"

    # consumers need cpr and nlohmann_json at build time
    requires = [
        "cpr/1.10.5",
        "nlohmann_json/3.11.3",
    ]

    # nothing to compile — no generators beyond CMakeDeps/CMakeToolchain
    generators = "CMakeDeps", "CMakeToolchain"

    exports_sources = "include/*", "CMakeLists.txt", "cmake/*"

    def layout(self):
        """
        Apply Conan's standard CMake directory layout to this recipe.
        
        Sets up the conventional source, build, and package directories used for CMake-based projects so the Conan recipe and CMake integration follow the expected layout.
        """
        cmake_layout(self)

    def package(self):
        # copy headers into the conan package store
        """
        Copy public header files into the package include directory.
        
        All files matching `*.hpp` are copied from the recipe's source_folder/include into package_folder/include so the packaged header-only library provides the expected include layout.
        """
        copy(self, "*.hpp",
             src=os.path.join(self.source_folder, "include"),
             dst=os.path.join(self.package_folder, "include"))

    def package_info(self):
        # header-only: no lib to link, just set the include path
        """
        Configure package metadata for consumers of this header-only library.
        
        Clears runtime and library directories so there are no binaries or linkable libraries, and sets the CMake integration properties:
        - `cmake_file_name` to "sf_voice"
        - `cmake_target_name` to "sf_voice::sf_voice"
        """
        self.cpp_info.bindirs  = []
        self.cpp_info.libdirs  = []
        self.cpp_info.set_property("cmake_file_name",   "sf_voice")
        self.cpp_info.set_property("cmake_target_name", "sf_voice::sf_voice")

    def package_id(self):
        # header-only packages are abi-independent
        """
        Mark the package as ABI-independent so it is treated as header-only.
        
        Clears the recipe's package identity information, ensuring the resulting package does not vary by settings such as compiler, OS, architecture, or build type.
        """
        self.info.clear()
