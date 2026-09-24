# Current State

This file is a compact snapshot of what is currently present in the repository. It is an index, not a specification. Update it when a major capability or constraint changes.

## Available foundation

- Language adapters under `Src/languages/` cover Python (primary), C#, C++, GDScript, Java, and Go. Comment adapters and Python-specific activity / timing / use-case helpers are present.
- Common analysis models / IR, call-graph analysis, class / object / package / component / deployment relations, partition, and CI workflow analysis are present under `Src/analyzers/` and `Src/models/`.
- Generators under `Src/generators/` include call graph, class / object / sequence / communication / package / component / deployment / state / timing / use-case diagrams, flowchart, responsibility table, and comment generation.
- Mermaid (and limited PlantUML) renderers for the above diagram types, responsibility tables, design-quality markdown, and CI graphs are present under `Src/renderers/`.
- Evaluators under `Src/evaluators/` provide deterministic CI quality checks and graph-based design-quality evaluation.
- Application process services, contracts, config, and data helpers live under `Src/process/` and `Src/data/`. Tkinter GUI entry is under `Src/ui/`.
- Automated tests cover call graph, comments (including multi-language adapters), diagrams, CI analysis/evaluation, design quality, contracts, context tooling, and packaging smoke.
- `app.py`, `run.bat`, packaging metadata, and GitHub Actions provide the current launch/build foundation.
- CI runs the full pytest suite on Python 3.11 / 3.12 / 3.13, architecture policy check, UPD Commander policy check, and self-analysis smoke.
- `next_issue.bat` and `pull_request.bat` provide the low-context Issue -> PR workflow.
- `context.bat` / `context.sh` expose repository profile, remote delta, compact diff, structure index, validation planning, policy checks, and Context Pack generation.

## Architecture direction

- Language-specific parsing stays behind language adapters.
- Common IR is the interchange boundary for language-independent analysis.
- Analysis / generation / evaluation and output rendering remain separate responsibilities.
- UI / Process / Data boundaries follow the Tomiya adaptation of UPD Commander Base Design.
- Commander and Messenger are orchestration/communication roles, not locations for real processing.
- Mermaid is the default diagram output; other formats belong behind renderer boundaries.

## Known limitations / active work

- Many UML generators and design evaluators exist in code but remain Issue-driven for completeness, GUI integration, and acceptance hardening.
- Python remains the first implementation target for most new analysis features; other language adapters are present at varying maturity (especially comment pipelines).
- GUI work is still an active area (file/operation compatibility, capability model); see open GUI Issues such as #73.
- The architecture policy checker intentionally distinguishes confirmed errors from review warnings so existing transitional code can be migrated without pretending every architectural rule is mechanically provable.
- Backend selection per language (AST / semantic engines) is still under evaluation (#60).

## Current highest-level goals

1. Stabilize Common IR and dependency boundaries.
2. Keep the project usable by humans and AI without full-repository rereads.
3. Build shared analysis primitives once, then reuse them across diagrams, tables, CI analysis, and design evaluation.
4. Keep language, UI, storage, and renderer dependencies at replaceable boundaries.

## Maintenance rule

Do not turn this file into a changelog. Keep only current capabilities, important constraints, known blockers, and near-term architectural state. Historical detail belongs in Git history, Issues, and PRs.
