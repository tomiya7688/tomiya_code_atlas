# GDScript AST / Semantic backend evaluation

Issue #133 evaluates GDScript analysis backends against the common conformance
fixture introduced by #131.

## Decision

Select **`tree-sitter-gdscript` as the primary GDScript syntax backend**.

Canonical package/repository used for the selection record:

```text
package: tree-sitter-gdscript
repository: PrestonKnopp/tree-sitter-gdscript
backend id: gdscript-tree-sitter
role: primary syntax backend
```

Do **not** treat tree-sitter as a complete semantic backend. Godot project
semantics must be layered separately behind the Language Adapter.

## Why tree-sitter-gdscript

The grammar explicitly models the GDScript constructs Tomiya needs for a
structural backend, including:

- `signal`
- `class_name`
- `extends`
- annotations such as `@export`
- typed variables and containers
- `await`
- lambdas
- match syntax
- Godot-specific StringName, NodePath and get-node syntax

`preload()` and `load()` are syntactically ordinary calls whose string/path
arguments can be extracted by the adapter and resolved later by a Godot project
resolver.

The observed release is **v6.1.0** and the repository has maintenance commits
observed through 2026-07-13. Its Python packaging exposes a native tree-sitter
grammar without requiring the user to install the Godot editor/engine.

## Candidate comparison

| Candidate | Syntax | Semantic/project data | Error recovery | Distribution | Decision |
| --- | --- | --- | --- | --- | --- |
| PrestonKnopp/tree-sitter-gdscript | Strong Godot-specific grammar | Syntax only | Tree-sitter tolerant parsing | Python/native assets can be bundled | **Primary syntax backend** |
| GDQuest/tree-sitter-gdscript | Same grammar family; active Godot-focused work | Syntax only | Same model | Same native binding model | Track as fork/mirror |
| Godot `GDScriptParser` | Authoritative and richest | Parser/type model integrated with Godot core/project | Native diagnostics | Strongly coupled to engine core/cache/resource classes | Not initial runtime backend |
| Current regex comment adapter | Very limited structural recognition | None | N/A | Trivial | Keep only for comment generation until replacement |

## Godot parser tradeoff

Godot's own parser is authoritative and exposes concepts such as
`PreloadNode`, `SignalNode`, `AwaitNode`, `ClassNode` and a rich
`DataType` model. That is attractive semantically.

However, it is an internal C++ parser coupled to Godot cache, Resource,
ScriptLanguage, Variant and other engine-core types. Shipping it as a Tomiya
helper would mean maintaining a substantial Godot-derived executable/library,
or requiring an installed Godot runtime. That conflicts with the current
self-contained Windows onedir distribution goal.

If later analysis needs exact Godot compiler semantics that cannot be recreated
reasonably from project files, a Godot-derived helper can be reevaluated behind
the already-versioned parser backend IPC contract.

## Semantic layering

The selected architecture is:

```text
.gd source
  -> tree-sitter-gdscript
  -> GDScript Language Adapter
       + Godot project resolver
         - res:// paths
         - preload/load targets
         - class_name registry
         - script inheritance
         - Node/Resource type catalogue
         - static signal connections where available
  -> Common IR
```

This keeps parser-native tree nodes and future Godot API objects below the
Language Adapter boundary.

## Conformance fixture

The GDScript fixture now includes:

- `class_name`
- `signal`
- `preload()`
- `load()`
- `Node` / `Resource` type references
- `@export`
- inheritance
- duplicate calls
- `await`
- lambda
- typed container

The fixture remains backend-neutral; no tree-sitter node names are added to the
Common IR golden.

## License and redistribution

- tree-sitter-gdscript: MIT.
- Godot Engine parser source: MIT.
- The selected tree-sitter path does not require users to install Godot.

The implementation follow-up must verify the Python/native grammar and
tree-sitter runtime inside the existing PyInstaller Windows **onedir** artifact.

## Follow-up

#133 is selection only. Replacing the current regex-only GDScript comment
adapter as an analysis backend requires a separate implementation issue. That
implementation must use the versioned `ParserBackend` boundary and #131
conformance fixture.

Sources checked on 2026-09-18:

- https://github.com/PrestonKnopp/tree-sitter-gdscript
- https://github.com/GDQuest/tree-sitter-gdscript
- https://github.com/godotengine/godot/blob/master/modules/gdscript/gdscript_parser.h
