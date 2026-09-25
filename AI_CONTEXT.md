# AI Context

Tomiya Code Atlas で AI が最初に読む小さい索引です。詳細仕様や履歴はここへ複製しません。

## Source of Truth
- Current task / priority / unfinished work: GitHub Issues. `.codex/next_issue.md` があればその1件を優先する。
- Implemented behavior: `Src/`, `tools/`, `tests/`
- Normative project rules: `specification/architecture-policy.md`
- Current capability / blocker snapshot: `docs/current_state.md`
- Responsibility routing: `docs/responsibility_map.md`
- Architecture explanation: `docs/architecture/upd_commander.md`
- Feature specs: `docs/specs/`
- Project operations: `docs/project_operations.md`
- Human overview: `README.md`

Generated `.codex/` packets, indexes, diagrams, reports, and summaries are derived indexes, not source of truth.

## Read First
1. `AI_CONTEXT.md`
2. `.codex/next_issue.md`（存在する場合）
3. `docs/responsibility_map.md`
4. current task に適用される `specification/` / feature spec
5. target source + matching tests

`AGENTS.md` contains stable agent-facing project rules; read it when the runtime does not already inject it or when a task touches architecture/workflow policy.

## Exploration Stop
Broad exploration を止める条件:
- Goal が分かる
- Required constraints が分かる
- Acceptance が分かる
- target source / tests / direct dependencies の working set が分かる
- 必要なら Out of Scope / Deferred が分かる

十分なら実装へ進み、不明点が発生したときだけ追加探索する。`context.bat exploration-stop` で Task Capsule の機械確認もできる。

## Context Priority
- P0: current task / Required / Acceptance
- P1: target source / matching tests
- P2: direct dependencies / active architecture rules
- P3: detailed references
- P4: history / unrelated Issues / generated artifacts

## Routing
- language adapter -> `Src/languages/`
- shared models / IR -> `Src/analyzers/`, `Src/models/`
- analysis -> `Src/analyzers/`
- logical generation -> `Src/generators/`
- output formatting -> `Src/renderers/`
- design evaluation -> `Src/evaluators/`
- Issue/PR/context workflow -> `tools/`, root `*.bat` / `*.sh`
- architecture rule -> `specification/architecture-policy.md`

Use `docs/responsibility_map.md` or `context.bat role-map` before broad file discovery.

## Architecture Constraints
- Language-specific AST / parser types stay behind language adapters.
- Common models / IR remain language-neutral; feature orchestration belongs elsewhere.
- Analyzer / Generator / Evaluator do not emit renderer syntax directly.
- Renderer does not parse source languages.
- UI / Process / Data responsibilities remain separated.
- Commander routes work; Messenger crosses boundaries; neither owns real processing.
- Confirmed architecture violations and review warnings must be distinguished.

## Working Rules
- Search first, read second. `context.bat search` / `path-find` are dependency-free bounded fallbacks.
- unrelated refactor を混ぜない。
- all docs / all Issues / repo history / full diff / full logs を無条件に読まない。
- changed files の次は changed symbols / direct dependencies へ絞る。
- summaries/indexes で不足する場合だけ原典へ戻る。
- deterministic analysis を LLM inference より優先できる箇所では優先する。
- remote競合があり得る場合は compact remote delta を先に確認する。
- generated / build / cache / large log は対象そのものが必要な場合だけ読む。
- failure logs は `compact-log` で error/warning/failure + bounded tail を優先する。
- 正確性をコンテキスト削減量より優先する。

## Validation
- change type に応じた smallest sufficient evidence を最初に選ぶ。
- `0 tests` / empty scan / unrelated smoke は成功根拠にしない。
- architecture-sensitive changes: `context.bat policy-check`
- tooling changes: targeted project-operation tests
- UI/visual acceptance: headless checks first, visual correctness がAcceptanceなら実画面確認も必要
- package/distribution changes: source checks + artifact/build smoke
- 実行できない範囲は `Unverified` として明示する。

## Low-context Commands
最短の作業開始は `prepare_work.bat` / `./prepare_work.sh`。最優先 Issue のTask Capsule、remote delta、Context Packを順に準備する。

`context.bat` / `./context.sh` の主なcommand:
- `profile` — repo規模 / context budget / hotspot候補
- `doc-index` — Markdown heading index
- `search` / `path-find` — bounded dependency-free search
- `role-map` / `truth-candidates` — 情報責務 / Source of Truth候補の索引
- `remote-delta` — ahead/behind/remote commits/files/bounded diff
- `compact-diff` — changed files / shortstat / commit subjects
- `structure-index` — Python symbols/importsの構造索引
- `validation-plan` — changed filesから検証をルーティング
- `policy-check` — compact architecture / UPD policy check
- `exploration-stop` — Goal/Required/Acceptance/Working Setの充足確認
- `compact-log` — failure/warningとbounded tailだけを残す
- `context-pack` — `.codex/context_pack.md` を生成

`next_issue.bat` は priority-first Task Capsuleを `.codex/next_issue.md` へ生成し、`pull_request.bat` は validation -> compact summary -> push -> PR を行う。

Safe remote update は `remote-delta --ff` を明示した場合のみ許可し、dirty/diverged state では停止する。

`ai-context-reducer` と `upd-commander-base-design` は設計参考元であり、Tomiya Code Atlas の実行時必須依存ではありません。
