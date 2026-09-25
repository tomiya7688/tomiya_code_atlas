# Project Operations

Tomiya Code Atlas uses GitHub Issues as the task ledger and defaults to `1 Issue ~= 1 PR`.

## Standard flow
## Remote latest first

### Build verification

`verify_build.bat` builds the wheel, source archive, and EXE, then runs both CLI outputs and static/test checks. `pull_request.bat` runs this verification before creating or updating the GitHub PR.

作業開始時は、リモートの最新状態を確認してから Issue の実装に入る。

```powershell
git status --short --branch
git fetch --prune origin
git rev-list --left-right --count main...origin/main
git pull --ff-only origin main
```

fast-forward できない場合は、ローカルコミットを確認してから `git merge --no-edit origin/main` で取り込む。競合は解消後にテストで確認し、force push でリモート履歴を上書きしない。

```text
Issue
 -> priority + Goal / Required / Acceptance
 -> prepare_work.bat / prepare_work.sh
 -> Task Capsule + remote delta + Context Pack
 -> Responsibility / Change Routing
 -> bounded search / structure index
 -> implementation
 -> targeted validation + policy check
 -> compact diff / compact log when useful
 -> pull_request.bat
 -> CI / review / merge
```

## Task selection
- P0: foundation / blocking correctness
- P1: major capabilities
- P2: extensions / productivity
- P3: future / low-priority improvements

Priority labels are authoritative; `[P0]`-`[P3]` title prefixes are a fallback. Explicit meta/roadmap/umbrella/index items are skipped by automatic selection when actionable work exists.

Issues should contain Goal, Required constraints, Acceptance, Priority, and Out of Scope/Deferred when needed. `next_issue.bat` extracts these into `.codex/next_issue.md`; the original Issue remains authoritative.

## Exploration control

Read in this order:

```text
Task Capsule / metadata
 -> Responsibility Map / file role map
 -> bounded search / structure index
 -> target source
 -> matching tests
 -> direct dependencies
 -> detailed docs only if needed
```

Stop broad exploration when Goal, Required, Acceptance, and the working set are sufficient. `context.bat exploration-stop` checks those fields mechanically; a passing result means broad discovery can stop, not that implementation is correct.

## Routing sources
- `docs/responsibility_map.md`: module/file ownership
- `specification/architecture-policy.md`: normative Required/Recommended/Advisory rules
- `docs/current_state.md`: current capabilities and blockers
- `docs/specs/`: feature details, only when routed by the task
- `context.bat role-map`: mechanical file-role index
- `context.bat truth-candidates`: likely authoritative locations without claiming authority automatically

Large documents should be entered through heading search (`doc-index`) or bounded text search rather than unconditional full reads.

## Search-first / Read-second
`context.bat search` and `path-find` are dependency-free bounded fallbacks. If `rg`, `fd`, IDE index, ctags, tree-sitter, or another stronger local tool is already available, it may be preferred; Tomiya must not require it for the basic workflow.

Search/index output selects source to read. It never replaces the source itself.

## Source Structure Index
`context.bat structure-index` emits a deterministic Python symbol/import index for `Src/` and `tools/`. It is a fallback routing index; richer Common IR/call/dependency analysis from Code Atlas should replace or augment it as the project matures.

Prefer bounded expansion:

```text
target symbol -> direct relations -> matching tests -> deeper graph only if needed
```

Fan-in/fan-out/cycles are impact-routing signals, not automatic design verdicts.

## Remote Delta First
Use `context.bat remote-delta` when another AI/chat/developer may have changed remote state. Inspect ahead/behind, commit subjects, changed files, shortstat, then the bounded diff excerpt. Read full changes only when the current task intersects them.

`remote-delta --ff` is explicit. It refuses dirty/diverged state and only performs a fast-forward.

## Context Pack
`context.bat context-pack` creates `.codex/context_pack.md` from the current Task Capsule, changed files, validation plan, compact diff, remote status, and exploration status. It is temporary derived context, not a specification; regenerate it instead of accumulating old packets.

## Repository profile / context budget
`context.bat profile` reports repository statistics, approximate full-read token cost, file types, and largest text files. Use it to identify context hotspots, not as a quality score.

## Compact change and log inspection
`compact-diff` returns changed-file status, shortstat, and commit subjects. This is the default handoff/PR summary input. Full diff is still used for actual review or ambiguity when needed.

`compact-log` keeps error/warning/failure lines plus a bounded tail. Successful logs should normally be summarized as pass/fail; expand raw logs only around a failure.

## Validation Routing
`context.bat validation-plan` maps changed files to useful evidence.

- logic: targeted tests -> regression -> broader suite when baseline permits
- architecture: targeted tests + `policy-check`
- GUI: headless behavior first; visual confirmation when visual correctness is Acceptance
- packaging: tests -> build -> artifact smoke when source checks are insufficient
- generated/bulk changes: reproducible transform / dry-run / representative validation
- random/time-dependent behavior: fixed input/seed/time bound + structured observation

A command that checks zero relevant targets is not evidence. Anything not checked is `Unverified`.

## Policy Routing
Normative rules live in `specification/architecture-policy.md`. `policy-check` checks only mechanically useful rules and distinguishes confirmed errors from warnings/review signals. Semantic architecture remains a targeted review task.

A Required-rule exception records rule, reason, scope, mitigation, removal/review condition, and source-of-truth reference.

## Information responsibilities
- README: human overview
- AI_CONTEXT: compact AI routing
- Current State: capabilities / blockers
- Responsibility Map: ownership routing
- docs: explanation / feature design
- specification: normative rules
- Issues: requirements / priority / incomplete work
- source + tests: implemented behavior
- `.codex/` and generated reports/diagrams: derived artifacts

Do not duplicate the same detailed specification across these surfaces.

## Commands
Windows: `context.bat command`; Linux/macOS: `./context.sh command`.

Main commands:
- discovery: `profile`, `doc-index`, `search`, `path-find`, `role-map`, `truth-candidates`, `structure-index`
- state/change: `remote-delta`, `compact-diff`, `context-pack`
- control: `exploration-stop`, `validation-plan`, `policy-check`, `compact-log`

`prepare_work.bat` / `prepare_work.sh` runs the normal preparation chain. `next_issue.bat` creates the priority-first Task Capsule. `pull_request.bat` performs the low-context validation/commit/push/PR flow.

## Adopted methods
The project adopts the applicable high-value methods from `ai-context-reducer` and UPD policy practice: compact AI entrypoint, Current State, Task Capsule/Context Pack, Search-first/Read-second, bounded search/path discovery, exploration stop conditions, explicit Out of Scope, priority-first actionable Issue selection, Task/Change Routing, Responsibility Map, file-role and source-of-truth indexing, Source Structure Index, context profile/budget/hotspots, Remote Delta First, compact diff/log handling, Validation Routing, Policy Routing/rule strength/scoped exceptions, deterministic-first structural analysis, generated/noisy-data exclusion, compact handoff reporting, and clear information responsibilities.

We do not maintain a second permanent full-repository analysis framework or an always-on giant call-graph cache. Code Atlas itself should become the richer structure-index provider as its shared analysis matures.

### UPD Commander Checker

The CI and development workflow installs the Python UPD Commander Checker from the upstream repository at the pinned commit in `tools/requirements-upd.txt`, then runs `upd-commander-check .`. This generic UI/Process/Data check complements the Tomiya-specific `context.bat policy-check`; the checker is development-only and is not included in runtime or distribution dependencies.
