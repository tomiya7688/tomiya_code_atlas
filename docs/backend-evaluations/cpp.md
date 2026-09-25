# C++ AST / Semantic backend evaluation

Issue #135 evaluates C++ analysis backends against the shared conformance
fixture.

## Decision

Select a **Clang LibTooling / full C++ AST helper** as the primary C++ backend.

```text
backend id: cpp-clang-tooling-helper
hosting: bundled native helper
protocol: Parser Backend Contract v1 JSON
```

Do not use libclang's C API as the primary semantic backend. The public
`clang-c/Index.h` documentation explicitly states that the C interface is
intended to remain relatively small/stable and will not expose all information
stored in Clang's C++ AST. Tomiya's template/overload/call-target goals need the
richer compiler AST/Sema layer.

tree-sitter-cpp remains an incomplete/editor-buffer syntax fallback candidate.

## Why full Clang tooling

C++ semantic interpretation depends heavily on:

- preprocessor macros
- include paths
- compile definitions/flags
- overload resolution
- templates and instantiations
- inheritance
- language standard/version
- platform/toolchain headers

A syntax-only parser cannot reliably reconstruct these rules. A Clang-based
helper can build actual translation units and use compiler semantic information.

## Fixture additions

The C++ fixture now contains:

- system and project includes
- macro expansion
- generic template class
- inheritance
- overloaded method
- duplicate call behavior
- async/future usage
- lambda/nested scope

These are capability probes. Clang Decl/Stmt kinds, SourceManager objects and
other compiler-native types remain below the Language Adapter/backend boundary.

## Candidate comparison

| Candidate | Templates/overloads | Preprocessor/includes | Incomplete source | Distribution | Decision |
| --- | --- | --- | --- | --- | --- |
| Clang LibTooling/full C++ AST helper | Compiler-native | Compiler-native | Diagnostics + partial recovery | Native helper + Clang assets | **Selected** |
| libclang C API | High-level subset | Available but intentionally limited API | Diagnostics/AST | libclang DLL | Not primary |
| tree-sitter-cpp | Syntax only | Syntactic preprocessing nodes | Strong | Native parser assets | Fallback |
| Current regex adapter | Comment heuristics only | No semantic model | N/A | Trivial | Not analysis backend |

## Project configuration

The helper should prefer `compile_commands.json` when available. That is the
best source of the actual flags used for a translation unit.

When no compilation database is available, Tomiya may use conservative
defaults and discovered include paths, but the result must be marked partial.
Missing headers or unresolved symbols must remain unresolved instead of being
fabricated.

## Distribution

The Windows application must not require an external LLVM/Clang installation.

Expected layout:

```text
backends/
  cpp/
    tomiya-cpp-backend.exe
    clang/llvm runtime DLLs or statically linked equivalents
    lib/clang/<version>/include/...   # Clang builtin resource headers
```

Bundling Clang itself does **not** solve all system/vendor header dependencies.
For example, exact MSVC standard-library semantics may depend on headers from a
toolchain/SDK that the analyzed project expects. Tomiya should still produce
partial structural results when possible and report missing configuration.

## License

LLVM/Clang is under **Apache-2.0 WITH LLVM-exception**.
tree-sitter-cpp is MIT.

## Follow-up

#135 records backend selection only. Implementation should be a separate issue
covering the native helper, compilation database support, Common IR mapping,
backend manifest and Windows onedir packaging.

Sources checked on 2026-09-18:

- https://github.com/llvm/llvm-project
- https://github.com/llvm/llvm-project/blob/main/clang/include/clang-c/Index.h
- https://github.com/tree-sitter/tree-sitter-cpp
