# Python AST / Semantic backend evaluation

Issue #132 evaluates Python parser backends against the backend conformance
fixture from #131.

## Decision

Keep **CPython stdlib `ast` as the primary Python parser backend** for the
current Common IR. Treat `symtable` as the standard-library semantic companion
for future symbol/scope IR work. Do not switch the primary path to LibCST or
tree-sitter-python yet.

The selected runtime backend ID is:

```text
python-stdlib-ast
```

`PythonStdlibBackend` implements Tomiya's versioned `ParserBackend` contract
and is the route used by `ApplicationService`.

## Candidate comparison

| Candidate | Syntax / AST | Symbol / semantic data | Incomplete source | Packaging | Common IR cost |
| --- | --- | --- | --- | --- | --- |
| stdlib `ast` + `symtable` | High semantic AST fidelity for the running Python version | Compiler scope/name binding through `symtable`; not full project type inference | Fails on invalid syntax | No extra dependency | Lowest |
| LibCST | Full-fidelity CST, formatting-preserving | Scope and qualified-name metadata; richer local metadata | Not selected as an error-recovery backend | Additional package/native build surface | Medium |
| tree-sitter-python | Strong incremental concrete syntax parser | No Python semantic layer by itself | Best candidate of the three for incomplete/editor buffers | Native/parser dependency | Medium-high |

Python's `symtable` exposes the compiler-created scope tables and identifier
binding information. LibCST provides metadata including scope and qualified
names, but its scope analysis is not complete arbitrary attribute/type
resolution. tree-sitter-python is useful when tolerant incremental parsing is
more important than semantic resolution.

## Why the primary backend stays stdlib

Tomiya's current Common IR consumes classes, functions, methods, inheritance,
calls, objects, state/timing facts and imports. The stdlib adapter already maps
those directly with no extra redistribution burden. Replacing it with a CST
backend would increase conversion and packaging complexity without improving
the fields currently consumed enough to justify the swap.

This is not a claim that stdlib AST is universally the best Python parser.
The choice is specifically for **the current Tomiya Common IR and packaged
desktop tool**.

## Known limits

1. Syntax support is bounded by the Python interpreter running Tomiya.
2. stdlib AST/symtable do not provide complete cross-module type or call-target
   inference.
3. Invalid/incomplete source is rejected; `PythonStdlibBackend` normalizes
   this as `unsupported_syntax`.
4. If editor-buffer/incomplete-source analysis becomes a product requirement,
   tree-sitter-python should be evaluated as a fallback path rather than
   silently replacing the semantic primary backend.

## Distribution and license

- CPython stdlib: already present in the packaged interpreter; no additional
  user runtime is required.
- LibCST: MIT licensed overall, with some PSF-derived files noted in its
  repository license. Current releases ship native/binary components, so a
  primary adoption would require explicit frozen-build verification.
- tree-sitter-python: MIT licensed; adoption would add the parser/binding native
  dependency surface.

No extra parser package is added by this decision, so the existing Windows
PyInstaller onedir packaging model remains unchanged.

## Verification

`tests/test_python_backend_selection.py` reuses the #131 conformance fixture,
checks the selected backend descriptor, verifies Common IR output, verifies the
stdlib symbol-table scope probe, and fixes the normalized syntax-error boundary.

Sources checked on 2026-09-18:

- Python `symtable`: https://docs.python.org/3/library/symtable.html
- LibCST: https://github.com/Instagram/LibCST
- tree-sitter-python: https://github.com/tree-sitter/tree-sitter-python
