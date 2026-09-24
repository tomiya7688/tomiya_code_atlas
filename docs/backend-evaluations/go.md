# Go AST / Semantic backend evaluation

Issue #137 evaluates Go parser/semantic backends against the common
conformance fixture introduced by #131.

## Decision

Select a **native Go helper built on the Go standard parser and type checker**:

- `go/parser`
- `go/ast`
- `go/token`
- `go/types`

```text
backend id: go-stdlib-types-helper
role: primary syntax + semantic
runtime: native helper executable
external Go installation: not required
```

## Why not make go/packages mandatory?

`golang.org/x/tools/go/packages` is excellent for package/workspace loading and
can return syntax plus complete `Types` / `TypesInfo`. However, its documented
default build tool is the **go command**.

Tomiya's distribution policy is to avoid silently requiring users to install a
compiler/SDK/toolchain. Therefore `go/packages` is an optional toolchain-aware
mode, not the mandatory runtime backend.

The self-contained path groups discovered project source packages itself and
uses `go/types` with a Tomiya-controlled importer/resolver.

## Semantic value of go/types

`go/types` performs:

- identifier/name resolution
- type deduction/checking
- interface satisfaction/type relationships
- generic type checking
- function/method selection information
- Defs/Uses mappings for symbols

This is the semantic information Tomiya needs for class/interface relations,
call resolution and design analysis.

## Candidate comparison

| Candidate | Syntax | Semantic resolution | Package loading | Distribution | Decision |
| --- | --- | --- | --- | --- | --- |
| stdlib parser + go/types helper | Official Go AST | Strong type checker | Tomiya-owned project resolver | Native executable | **Selected** |
| go/packages | Official syntax/types | Strong | Excellent module/build-aware loading | Normally invokes `go` command | Optional enhanced mode |
| gopls | Rich workspace analysis | Strong | Rich workspace model | Larger protocol/toolchain surface | Not primary |
| tree-sitter-go | Incremental syntax | None by itself | Syntax only | Small native parser | Fallback |
| current regex comment adapter | Limited | None | None | Trivial | Comment-only |

## Packaging

The helper is compiled in CI and distributed as:

```text
backends/go/
└─ tomiya-go-backend.exe
```

A Go binary carries its runtime, so users do not install Go merely to execute
the helper. The helper uses Parser Backend Contract v1 over stdin/stdout.

## Project/module strategy

Self-contained mode:

1. Tomiya discovers `.go` source files.
2. Files are grouped by directory/package declaration.
3. Project-local imports are resolved from discovered source packages.
4. Vendor/source trees are used when available.
5. Missing external packages remain **unresolved**; Tomiya must not invent
   symbols.

Optional toolchain-aware mode may use `go/packages` when a compatible Go
toolchain is intentionally available. The application must not silently change
from self-contained mode into a system-toolchain requirement.

## Conformance fixture

The Go fixture now includes:

- package declaration
- standard and project import
- generic interface
- generic struct
- embedded base struct
- methods
- duplicate calls
- goroutine/channel/select
- closure/nested function
- context cancellation path

## License

Go and x/tools use the Go BSD-style license. tree-sitter-go is MIT.

## Known limits

- Build tags, GOOS/GOARCH and exact module selection can affect semantic output.
- Self-contained mode may not resolve external modules whose source/export data
  is not available.
- Parser syntax coverage follows the Go version used to build the helper.
- Full toolchain-aware module semantics can be added through the optional
  `go/packages` mode.

## Follow-up

#137 is selection only. A separate implementation issue should create the Go
helper, custom project importer, Common IR conversion, backend manifest entry,
and Windows onedir packaging.

Sources checked on 2026-09-18:

- https://github.com/golang/go
- https://github.com/golang/tools/tree/master/go/packages
- https://github.com/tree-sitter/tree-sitter-go
