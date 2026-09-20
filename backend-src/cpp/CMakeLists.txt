cmake_minimum_required(VERSION 3.20)
project(kadoka_cpp_backend LANGUAGES CXX)

find_package(LLVM REQUIRED CONFIG)
find_package(Clang REQUIRED CONFIG)

add_executable(kadoka-cpp-backend src/main.cpp)
target_compile_features(kadoka-cpp-backend PRIVATE cxx_std_17)
target_include_directories(
    kadoka-cpp-backend
    SYSTEM PRIVATE
    ${LLVM_INCLUDE_DIRS}
    ${CLANG_INCLUDE_DIRS}
)
target_compile_definitions(kadoka-cpp-backend PRIVATE ${LLVM_DEFINITIONS})

llvm_map_components_to_libnames(LLVM_LIBS Support Option)

if(TARGET clang-cpp)
    target_link_libraries(kadoka-cpp-backend PRIVATE clang-cpp ${LLVM_LIBS})
else()
    target_link_libraries(
        kadoka-cpp-backend PRIVATE
        clangTooling
        clangToolingCore
        clangFrontend
        clangSerialization
        clangDriver
        clangParse
        clangSema
        clangAnalysis
        clangASTMatchers
        clangAST
        clangLex
        clangBasic
        ${LLVM_LIBS}
    )
endif()
