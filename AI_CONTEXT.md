# AI向け作業索引

Tomiya Code AtlasでAIが最初に読む簡潔な索引です。詳細仕様や履歴はここへ複製しません。

## 正本
- 現在の作業、優先度、未完了内容: GitHub Issues。`.codex/next_issue.md` があれば、そのIssueを優先する。
- 実装済みの挙動: `Src/`、`tools/`、`tests/`
- 規範ルール: `specification/architecture-policy.md`
- 現在の機能と阻害要因: `docs/jp/現状.md`
- 責務routing: `docs/jp/責務マップ.md`
- アーキテクチャ説明: `docs/jp/構成/UPDコマンダー適用.md`
- 機能仕様: `docs/jp/機能仕様/`
- 開発運用: `docs/jp/開発運用.md`
- 英語版と利用者ガイド: [`docs/en/README.md`](docs/en/README.md)（日本語正本から作成）
- 人向け概要: `README.md`

生成された `.codex/` のpacket、index、図、report、summaryは派生indexであり、正本ではありません。

## 最初に読むもの
1. `AI_CONTEXT.md`
2. 存在する場合は `.codex/next_issue.md`
3. `docs/jp/責務マップ.md`
4. 現在のtaskに適用される`specification/`または機能仕様
5. 対象ソースと対応test

`AGENTS.md`には安定したagent向けルールがあります。実行環境が内容を自動で渡さない場合、またはアーキテクチャ・作業手順に関わる場合に確認してください。

## 探索を止める条件
- Goalが分かっている
- Required制約が分かっている
- Acceptanceが分かっている
- 対象ソース、test、直接の依存先が特定できている
- 必要な場合、対象外や先送り範囲が分かっている

十分に分かったら実装へ進み、新たな不明点が生じた場合だけ調査を追加します。`scripts/dev/context.bat exploration-stop` でTask Capsuleの充足状況を機械的に確認できます。

## Contextの優先度
- P0: 現在のtask、Required、Acceptance
- P1: 対象ソースと対応test
- P2: 直接の依存先と適用されるarchitecture rule
- P3: 詳細資料
- P4: 履歴と無関係なIssue・生成物

## 対象へのrouting
- 言語adapter -> `Src/languages/`
- 共通model/IR -> `Src/analyzers/`、`Src/models/`
- 解析 -> `Src/analyzers/`
- 論理出力の生成 -> `Src/generators/`
- 出力形式の処理 -> `Src/renderers/`
- 設計評価 -> `Src/evaluators/`
- Issue/PR/context workflow -> `tools/`、`scripts/dev/`
- Windows build入口 -> rootの `build_exe.bat` / `run_dist.bat`、補助file -> `scripts/build/`
- architecture rule -> `specification/architecture-policy.md`

広くfileを探す前に `docs/jp/責務マップ.md` または `scripts/dev/context.bat role-map` を使います。

## アーキテクチャ制約
- 言語固有のAST/parser型は言語adapterの内側に置きます。
- 共通model/IRは言語非依存とし、機能の調整処理を持たせません。
- Analyzer/Generator/Evaluatorはrenderer固有の構文を直接出力しません。
- Rendererはソース言語を解析しません。
- UI / Process / Dataの責務を分離します。
- Commanderは処理を振り分け、Messengerは境界を越えて情報を運びます。どちらにも実処理を置きません。
- 確認済みの違反と、warning/review候補を区別します。

## 作業ルール
- 先に検索し、その後必要な内容を読みます。`scripts/dev/context.bat search` / `path-find` は追加依存なしのfallbackです。
- 無関係なrefactorを混ぜません。
- すべてのdocs、Issue、repository履歴、full diff、full logを無条件に読みません。
- fileを変更したら、関連symbolと直接の依存先を確認します。
- summary/indexが不十分な場合だけ原典へ戻ります。
- 決定的な解析が可能なら、LLM推論より優先します。
- remoteの競合があり得る場合、広く読み直す前にcompact deltaを確認します。
- 生成物、build、cache、大きなlogは、taskで必要な場合だけ読みます。
- 失敗logは `compact-log` でerror/warning/failureと範囲を絞った末尾を確認します。
- context削減量より正確性を優先します。

## 検証
- 変更内容に合った、必要十分で最小の検証から始めます。
- `0 tests` / 空のscan / 無関係なsmokeを成功の根拠にしません。
- アーキテクチャに関わる変更: `scripts/dev/context.bat policy-check`
- toolingの変更: 対象となるproject-operation test
- UI/画面のAcceptance: まずheadless testを実行し、見た目の確認が条件なら実画面も確認
- package/配布物の変更: source testに加え、build/artifact smokeを実行
- 実行しなかった検証は `Unverified` と記載

## 低contextで使うcommand
最短の準備手順は `scripts/dev/prepare_work.bat` / `./scripts/dev/prepare_work.sh` です。優先IssueのTask Capsule、remote delta、Context Packを順に準備します。

`scripts/dev/context.bat` / `./scripts/dev/context.sh` の主なcommand:
- `profile` — repository規模とcontext使用量
- `doc-index` — Markdown見出しのindex
- `search` / `path-find` — 追加依存なしの範囲指定検索
- `role-map` / `truth-candidates` — 責務と正本候補の案内
- `remote-delta` — ahead/behind、remote変更file、差分概要
- `compact-diff` — 変更file、shortstat、commit概要
- `structure-index` — Python symbol/importの構造index
- `validation-plan` — 変更fileに応じた検証案内
- `policy-check` — architecture/UPD policyの確認
- `exploration-stop` — Goal/Required/Acceptance/Working Setの充足確認
- `compact-log` — failure/warningと範囲を絞った末尾
- `context-pack` — `.codex/context_pack.md` を生成

`scripts/dev/next_issue.bat` は優先順のTask Capsuleを `.codex/next_issue.md` に作成します。`scripts/dev/pull_request.bat` は検証、簡潔な要約、push、PR作成を行います。

安全なremote更新には `remote-delta --ff` の明示指定が必要です。working treeに変更がある場合や、履歴が分岐している場合は停止します。

`ai-context-reducer` と `upd-commander-base-design` は設計上の参考資料です。Tomiya Code Atlasの実行時必須依存ではありません。
