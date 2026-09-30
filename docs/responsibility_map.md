# Responsibility Map

Use this map to route a task before opening implementation files. Keep each responsibility short; detailed behavior belongs in source, tests, or feature specs.

| Area | Responsibility | Normal dependencies / notes |
|---|---|---|
| `app.py`, `run.bat` | launch surface / future UI entry | call application/process services; do not absorb analysis or storage logic |
| `setup.bat`, `build*.bat`, `run*.bat`, `verify_build.bat` | Windows setup, build, launch, and distribution verification | `setup.bat` prepares `.venv`; source and packaged launchers stay explicit |
| `scripts/dev/` | Windows/Linux/macOS Issue, context, reducer, and PR helper entry points | scripts change to repository root before resolving `tools/` paths |
| `Src/languages/` | language-specific parsing and adapters | external parser/AST details stop here; produce shared models / IR |
| `Src/analyzers/` | deterministic relationship / flow / graph analysis | consume shared models; must not depend on renderer syntax |
| `Src/models/` | passive shared data contracts | keep cross-layer data small and serialization-friendly; avoid feature orchestration |
| `Src/generators/` | transform analyzed information into logical outputs | avoid output-format syntax; legacy language coupling is a migration warning |
| `Src/renderers/` | serialize logical results to Mermaid / text / future formats | no language parsing; keep format-specific concerns here |
| `Src/evaluators/` | design / code-quality evaluation | consume language-neutral analysis results where practical |
| `tools/next_issue.py` | choose one priority Issue and create a Task Capsule | deterministic; no LLM dependency |
| `tools/create_pr.py` | validate, summarize, push, and create a compact PR | full diff only when ambiguity/failure requires it |
| `tools/context_tool.py` | repository/context indexing and compact policy/validation helpers | generated outputs are indexes, never source of truth |
| `docs/specs/` | feature-level design / acceptance context | read only the spec routed by the current task |
| `docs/architecture/` | explanatory architecture guidance | background and mapping; normative rules live in `specification/` |
| `docs/project_operations.md` | Issue/PR/context workflow | stable operational policy |
| `docs/current_state.md` | compressed current capabilities / blockers | update when major state changes; not a changelog |
| `specification/` | normative Required / Recommended project rules | exceptions must be explicit and scoped |
| `tests/` | executable behavior / compatibility evidence | test names should route back to affected responsibility areas |
| `.github/workflows/` | CI evidence and packaging gates | compact, deterministic validation preferred |

## Routing rule

Start with the row matching the task, then read:

```text
responsibility area
  -> matching tests
  -> direct dependencies
  -> relevant specification only if needed
```

Do not open every area because a task might eventually interact with it.

## Architecture signal

If a row can no longer be described in one short responsibility sentence, or a new module clearly owns two unrelated responsibilities, treat that as a design-review signal. It is not an automatic split requirement, but it should trigger a targeted architecture check.

## Update rule

Update this map in the same change set when a major module is added, split, merged, or changes architectural ownership. Do not list individual helper functions here.
