# Java AST / Semantic backend evaluation

Issue #136 evaluates Java parser/semantic backends against the common
conformance fixture introduced by #131.

## Decision

Select **JavaParser + JavaSymbolSolver** as the primary Java syntax and semantic
backend, hosted in a bundled Java helper.

```text
backend id: java-javaparser-symbol-solver-helper
role: primary syntax + semantic
transport: Parser Backend Contract v1 JSON
runtime: bundled private JVM
```

The user must not be required to install a JDK, JRE, Maven or Gradle merely to
run Tomiya analysis.

## Why JavaParser + Symbol Solver

Tomiya needs more than Java syntax. Class/package diagrams and call/relationship
analysis benefit from resolved declarations, imports, inheritance/interfaces,
generic types, overloads and lambdas.

JavaParser's project advertises modern Java parsing together with advanced
analysis functionality, and its Symbol Solver is specifically intended for
resolving declarations and types. This is a substantially better fit than a
syntax-only tree-sitter backend while keeping the standalone helper surface
smaller than Eclipse JDT.

The latest observed JavaParser release for this evaluation is **3.28.2**
(2026-05-31). The repository remained active in September 2026.

## Candidate comparison

| Candidate | Syntax | Symbols / overloads | Generic / lambda | Packaging | Decision |
| --- | --- | --- | --- | --- | --- |
| JavaParser + Symbol Solver | Modern Java AST | Project/source/jar-aware resolution | Strong semantic support | Helper JAR + private JVM | **Selected** |
| Eclipse JDT Core | Compiler-grade | Very strong binding/compiler model | Very strong | Larger Eclipse/JDT integration surface | Not primary |
| tree-sitter-java | Strong incremental syntax | None by itself | Syntax only | Small native parser | Fallback for malformed buffers |
| Current regex comment adapter | Limited | None | None | Trivial | Comment generation only |

## Distribution

The implementation follow-up should bundle:

```text
backends/java/
├─ tomiya-java-backend.jar
└─ runtime/
   └─ bin/java.exe
```

The runtime should be a redistributable private OpenJDK runtime assembled during
the build. A minimized `jlink` runtime may be used after dependency verification,
but trimming must not be allowed to silently break Symbol Solver behavior.

The Python host launches the helper through the already-defined Parser Backend
Contract v1. stdout is protocol-only; diagnostics go to stderr.

## Classpath / project semantics

The helper should resolve, when available:

- project source roots
- imports and packages
- interfaces and inheritance
- overloaded methods
- generic type parameters and arguments
- lambda/function-interface targets
- dependency JARs/classpath entries

Maven or Gradle metadata may help discover dependencies, but Tomiya should not
require Maven/Gradle to be installed just to parse basic Java source. Missing
external dependencies must become explicit unresolved facts rather than invented
symbols.

## License

JavaParser is offered under LGPL or Apache License 2.0 terms. Tomiya should use
the **Apache-2.0** option and preserve required notices.

Eclipse JDT Core is EPL-2.0. tree-sitter-java is MIT.

## Conformance fixture

The Java fixture now contains:

- `package`
- multiple `import` declarations
- interface implementation
- class inheritance
- generic class/interface
- overloaded helper methods
- duplicate method calls
- lambda
- CompletionStage/CompletableFuture asynchronous-style flow

No JavaParser/JDT/tree-sitter node names are added to the Common IR golden.

## Follow-up

#136 is selection only. A separate implementation issue should build the Java
helper, integrate Symbol Solver, normalize output to Common IR, bundle the
private JVM in the Windows onedir artifact and run the #131 fixture in CI.

Sources checked on 2026-09-18:

- https://github.com/javaparser/javaparser
- https://github.com/eclipse-jdt/eclipse.jdt.core
- https://github.com/tree-sitter/tree-sitter-java
