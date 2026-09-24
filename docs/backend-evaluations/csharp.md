# C# AST / Semantic backend evaluation

Issue #134 evaluates C# analysis backends against the conformance fixture from
#131.

## Decision

Select **Roslyn / Microsoft.CodeAnalysis.CSharp** as the primary C# syntax and
semantic backend.

```text
backend id: csharp-roslyn-helper
hosting: bundled helper executable
protocol: Parser Backend Contract v1 JSON
runtime: self-contained .NET publish
```

tree-sitter-c-sharp remains a possible syntax fallback for malformed editor
buffers, but it is not the semantic primary backend.

## Why Roslyn

Tomiya needs more than a syntax tree for C#:

- symbol identity
- overload resolution
- base/interface relationships
- generic type parameters and substitutions
- method/call target binding
- accessibility
- diagnostics for incomplete or invalid source

Roslyn is the C# compiler platform itself and exposes both syntax trees and the
`SemanticModel` / symbol model used for compiler binding. This avoids
reimplementing C# overload and type-resolution rules in Python.

tree-sitter-c-sharp is a strong incremental syntax parser and remains useful as
a future recovery/fallback option, but its own project explicitly encounters
ambiguities that require semantic information to resolve. That is exactly the
class of information Tomiya needs for reliable call/class diagrams.

## Fixture additions

The C# fixture now includes:

- generic `Worker<T>`
- inheritance
- `IWorker<T>` interface implementation
- overloaded `run` methods
- duplicate call sequence
- async/await
- local function / nested scope

These are capability probes. Backend-specific Roslyn syntax kinds and symbol
objects are not added to the Common IR golden.

## Helper architecture

Roslyn runs in a bundled .NET helper rather than in the Python process.

```text
C# source/project
  -> backends/csharp/tomiya-csharp-backend.exe
      -> Roslyn syntax trees + compilation + SemanticModel
      -> normalized wire DTO / Common IR facts
  -> Python Language Adapter boundary
  -> Common IR
```

The helper uses the already-defined Parser Backend Contract v1. stdout is
reserved for protocol messages; diagnostics/logging go to stderr. timeout,
protocol mismatch and unsupported inputs are normalized by the host contract.

### Runtime distribution

Publish as a **self-contained win-x64 folder**, not as a single file:

```text
backends/
  csharp/
    tomiya-csharp-backend.exe
    *.dll
    *.json
    .NET runtime files
```

The publish command is expected to use the equivalent of:

```text
dotnet publish -c Release -r win-x64 --self-contained true
```

`PublishSingleFile` and trimming remain disabled initially. This minimizes
Roslyn reflection/loading surprises and matches Tomiya's existing onedir
distribution model. A self-contained .NET publish includes the required .NET
runtime in the deployment folder, so the target machine does not need a
separately installed .NET runtime.

## Project semantics without an SDK requirement

The initial helper should use `CSharpCompilation` directly instead of making
`MSBuildWorkspace` a runtime requirement.

- Tomiya project discovery supplies C# source files.
- The helper creates syntax trees and a compilation.
- Framework metadata references are obtained from the helper's own runtime.
- Project/source references are resolved from discovered project inputs.
- Unity or third-party assemblies are used when project-resolvable; otherwise
  affected symbols remain explicitly unresolved.

This keeps the runtime self-contained and avoids requiring the user to install
the .NET SDK or MSBuild.

## Candidate comparison

| Candidate | Syntax | Symbols/overloads/interfaces/generics | Incomplete source | Packaging | Decision |
| --- | --- | --- | --- | --- | --- |
| Roslyn | Compiler-authoritative | Compiler-native semantic binding | Syntax tree + diagnostics | Self-contained .NET helper folder | **Selected** |
| tree-sitter-c-sharp | Strong incremental grammar | No semantic binding by itself | Strong | Native parser assets | Fallback candidate |
| Current regex adapter | Comment-oriented heuristics | None | N/A | Trivial | Not an analysis backend |

## License

- Roslyn: MIT.
- tree-sitter-c-sharp: MIT.

## Implementation status

The selected backend is integrated by #150.

- helper source: `backend-src/csharp/`
- Python host: `Src/languages/csharp_backend.py`
- shared subprocess protocol host: `Src/languages/helper_backend.py`
- runtime path: `backends/csharp/tomiya-csharp-backend.exe`
- build: self-contained `win-x64`, not single-file, not trimmed
- Linux CI: Roslyn helper build + #131 fixture semantic smoke
- Windows CI: publish + PyInstaller onedir + frozen EXE C# backend smoke

The user-facing distribution does not require a separately installed .NET SDK
or runtime.

Sources checked on 2026-09-18:

- https://github.com/dotnet/roslyn
- https://github.com/tree-sitter/tree-sitter-c-sharp
- https://learn.microsoft.com/dotnet/core/deploying/
