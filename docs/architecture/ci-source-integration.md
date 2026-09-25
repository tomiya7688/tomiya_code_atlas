# CI / Source Analysis Integration

Tomiya Code Atlas links CI structure to source analysis through conservative, language-neutral path hints. The CI analyzer does not open source files, resolve imports, or invoke language parsers.

## Pipeline

```text
CI provider configuration
  -> provider analyzer
  -> CIWorkflow / CIJob / CIStep
  -> CIStep.path_hints
  -> collect_ci_path_hints()
  -> project file discovery / existence filtering
  -> language detection
  -> Language Adapter
  -> Common IR / graph / diagram / evaluator
```

## Path hint contract

Each `CIStep` may contain zero or more `path_hints`. A hint is evidence found in the CI configuration, not a guarantee that the path exists or that it is source code.

Examples:

- `pytest tests/unit/test_service.py` -> `tests/unit/test_service.py`
- `ruff check Src tests` -> `Src`, `tests`
- `working-directory: ./backend` -> `backend`
- `uses: ./.github/actions/setup` -> `.github/actions/setup`

`collect_ci_path_hints()` preserves the originating job and step through `CIPathHint(job, step, path)`.

## Responsibility boundary

The CI analyzer is responsible for:

- provider syntax parsing;
- jobs, steps and `needs` relationships;
- command/action metadata;
- conservative path-hint extraction.

The CI analyzer is **not** responsible for:

- filesystem existence checks;
- glob expansion;
- deciding whether a path is source, test, generated output or tooling;
- language detection;
- source parsing;
- Common IR construction for source code.

Those responsibilities stay in the existing project discovery and Language Adapter boundaries. This keeps CI analysis provider-specific only at the input edge and allows the same later source-analysis pipeline to be reused.

## Future reverse tracing

A later project-level trace can resolve path hints against the selected repository and construct links such as:

```text
changed source
  -> matching CI path hint / project scope
  -> CI step
  -> CI job
  -> downstream needs edges
  -> package / deploy artifact
```

The resolver should keep unresolved or ambiguous hints as such rather than inventing a definite link.
