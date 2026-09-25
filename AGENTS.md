# Tomiya Code Atlas — Agent Guide

## Start here
Read `AI_CONTEXT.md` first. It is the compact routing index.

If `.codex/next_issue.md` exists, treat that Issue as the active task. Do not fetch unrelated Issues unless the selected task requires them. Use `docs/responsibility_map.md` to find the initial source/test area and `specification/architecture-policy.md` for normative rules.

## Purpose
Tomiya Code Atlas is a source-code analysis toolkit that generates diagrams, tables, comments, evaluations, and CI-oriented analysis results while keeping language-specific parsing, shared analysis, and output formats replaceable.

## Primary analysis targets
- Python
- GDScript
- C#
- C++
- Java
- Go

Python is normally the first implementation target for new analysis features. Do not make shared architecture dependent on one analysis language.

## Repository responsibility boundaries
- `Src/languages/` — language-specific parsing / adapters
- `Src/analyzers/` — deterministic relationship / graph / flow analysis
- `Src/models/` — passive shared data contracts
- `Src/generators/` — logical output generation
- `Src/renderers/` — Mermaid/text/future format serialization
- `Src/evaluators/` — design/code-quality evaluation
- `tools/` — local Issue/PR/context helpers
- `docs/` — explanatory architecture, current state, routing, feature design
- `specification/` — normative project rules

Detailed ownership is in `docs/responsibility_map.md`.

## Required architecture rules
- Keep language-specific AST/parser/library types inside the language boundary.
- Shared IR/models must remain language-neutral and should not own orchestration/evaluation/rendering behavior.
- Keep analysis logic independent from UI and renderer syntax.
- Prefer deterministic static analysis when information can be obtained mechanically.
- Separate LLM-assisted / inferred information from confirmed deterministic results.
- Reuse shared analysis results across multiple generators/evaluators instead of reparsing independently.
- Renderers must not parse source languages.
- UPD application boundary: UI handles presentation/input, Process handles orchestration/analysis flow, Data handles I/O/persistence/external-data details.
- Commander routes work and stays thin; Messenger crosses a boundary and contains no domain processing.

The complete Required/Recommended/Advisory policy is `specification/architecture-policy.md`.

## Output policy
- Diagram output defaults to Mermaid.
- Comment generation writes comments into source code.
- Tables should use simple machine-readable structures where practical.
- PlantUML and future formats belong behind renderer boundaries.
- Class diagrams default to caller-centered grouping; heavily referenced classes may get dedicated callee-centered views.

## Low-context workflow
The current `ai-context-reducer` is a development-only dependency. Use `reducer.bat setup` on Windows or `./reducer.sh setup` on Unix-like systems to clone/update the latest reducer into `.dev/ai-context-reducer` and analyze this repository. The `.dev/` checkout is intentionally ignored and must never be included in product builds or release artifacts.

`prepare_work.bat` invokes the reducer first, then falls back to the repository-local context helpers if reducer setup is unavailable.

Before general Issue work, use `next_issue.bat` to select one priority task and create a structured Task Capsule.

For concurrent work, use `context.bat remote-delta` (or `./context.sh remote-delta`) before broad re-reading. It reports ahead/behind, commit subjects, changed files, shortstat, and a bounded diff excerpt. `--ff` is explicit and only allows a clean fast-forward.

Useful commands:
- `reducer.bat update` / `./reducer.sh update`
- `reducer.bat analyze` / `./reducer.sh analyze`
- `context.bat profile`
- `context.bat doc-index`
- `context.bat structure-index`
- `context.bat compact-diff`
- `context.bat validation-plan`
- `context.bat policy-check`
- `context.bat context-pack`

Use `pull_request.bat` after completing an Issue. It validates, commits, pushes, and creates a compact PR without requiring a full-diff read merely to write the summary.

## Context discipline
- Search first, read second.
- Stop broad exploration once Goal / Required / Acceptance / working set are sufficient.
- Prefer current-task source -> matching tests -> direct dependencies -> detailed docs when needed.
- Do not load all feature specs, all Issues, repository history, full logs, or full diffs by default.
- Treat generated Context Packs/indexes/diagrams as routing aids, not source of truth.
- Use bounded symbol/graph/diff expansion when available.
- Keep unrelated refactors and deferred features out of the active Issue.
- Return to original source/tests/docs when an index or summary is insufficient.
- Report unexecuted validation as `Unverified`.

## Validation discipline
Choose evidence by change type. Targeted checks come first; completion gates expand only as needed. A passing command that inspected zero relevant targets is not evidence. Visual acceptance requires visual confirmation when the UI itself is the requirement. Packaging changes should validate the generated artifact when source-only checks are insufficient.

CI also runs the compact architecture policy checker. Confirmed violations are errors; uncertain architectural signals remain warnings/review candidates.

## Information responsibilities
- README: human overview / setup
- AI_CONTEXT: compact AI routing
- Current State: current capabilities / blockers
- Responsibility Map: file/module routing
- docs: explanation / feature design
- specification: normative policy
- Issues: requirements / priority / unfinished work
- source + tests: implemented behavior

Do not duplicate the same detailed specification across these surfaces.

## License
MIT
