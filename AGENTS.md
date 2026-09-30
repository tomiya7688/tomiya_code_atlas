# Tomiya Code Atlas — エージェント向け手引き

## 最初に読む
まず `AI_CONTEXT.md` を読んでください。作業対象への簡潔な案内です。

`.codex/next_issue.md` があれば、そのIssueを作業対象として扱います。選択されたIssueに必要な場合を除き、無関係なIssueを取得しないでください。最初に読むソースとテストを `docs/jp/責務マップ.md` で絞り、規範ルールは `specification/architecture-policy.md` で確認してください。

## 目的
Tomiya Code Atlasは、言語固有の解析、共通解析、出力形式を交換可能に保ちながら、図、表、コメント、評価、CI解析結果を生成するソースコード解析ツールです。

## 主な解析対象
- Python
- GDScript
- C#
- C++
- Java
- Go

新しい解析機能は、通常Pythonから実装します。共通アーキテクチャを特定の解析言語に依存させないでください。

## リポジトリ内の責務境界
- `Src/languages/` — 言語固有の解析とadapter
- `Src/analyzers/` — 決定的な関係・graph・flow解析
- `Src/models/` — 受動的な共通data契約
- `Src/generators/` — 論理出力の生成
- `Src/renderers/` — Mermaid/text/将来形式へのシリアライズ
- `Src/evaluators/` — 設計・コード品質の評価
- `tools/` — ローカルIssue/PR/context補助
- `docs/jp/` — アーキテクチャ、現状、routing、機能設計の日本語正本
- `docs/en/` — 日本語正本から派生する英訳
- `specification/` — 規範となるプロジェクトルール

詳細な所有関係は `docs/jp/責務マップ.md` に記載します。

## Windowsのbuild入口
- リポジトリrootには利用者が直接実行する `build_exe.bat` と `run_dist.bat` を置きます。
- setup、Python packageのbuild、ソース起動、全体検証は `scripts/build/` に置き、リポジトリrootを基準にpathを解決します。
- 入口を変更した場合は、workflowと文書の参照先も揃えてください。

## 必須アーキテクチャルール
- 言語固有のAST/parser/library型は言語境界の内側に保ちます。
- 共通IR/modelは言語非依存に保ち、処理調整・評価・描画の振る舞いを持たせません。
- 解析処理をUIやrendererの構文から独立させます。
- 機械的に取得できる情報には、決定的な静的解析を優先します。
- LLM支援・推定情報と、決定的に確認した結果を分けます。
- 同じ解析を繰り返さず、複数のgenerator/evaluatorで共通の解析結果を再利用します。
- Rendererでソース言語を解析しません。
- UPD境界: UIは表示と入力、Processは処理の調整・解析の流れ、Dataは入出力・永続化・外部dataの詳細を扱います。
- Commanderは処理を振り分けて薄く保ちます。Messengerは境界を越えて情報を運び、領域処理を持ちません。

Required/Recommended/Advisoryの規則は `specification/architecture-policy.md` を参照してください。

## 出力方針
- 図の既定出力形式はMermaidです。
- コメント生成はソースコードへコメントを書き込みます。
- 表は、可能なら単純で機械処理しやすい形式にします。
- PlantUMLや将来の形式はrenderer境界の背後へ追加します。
- クラス図は呼び出し元を中心にまとめ、多数から参照されるクラスは呼び出し先中心の図へ分ける場合があります。

## 低contextでの作業手順
現在の `ai-context-reducer` は開発時だけに使う依存です。Windowsでは `scripts/dev/reducer.bat setup`、Unix系では `./scripts/dev/reducer.sh setup` を使い、最新のreducerを `.dev/ai-context-reducer` に取得・更新して、このrepositoryを解析します。`.dev/` は意図的にignoreしており、製品buildやrelease成果物へ含めてはいけません。

`scripts/dev/prepare_work.bat` は最初にreducerを実行し、利用できない場合はrepository内のcontext補助機能へ切り替えます。

通常のIssue作業を始める前に `scripts/dev/next_issue.bat` で優先Issueを選び、構造化Task Capsuleを作成します。

並行作業がある場合は、広く読み直す前に `scripts/dev/context.bat remote-delta`（または `./scripts/dev/context.sh remote-delta`）を実行します。ahead/behind、commit概要、変更file、shortstat、範囲を絞ったdiff抜粋を確認できます。`--ff` は明示的な指定で、cleanなfast-forwardだけを許可します。

よく使うcommand:
- `scripts/dev/reducer.bat update` / `./scripts/dev/reducer.sh update`
- `scripts/dev/reducer.bat analyze` / `./scripts/dev/reducer.sh analyze`
- `scripts/dev/context.bat profile`
- `scripts/dev/context.bat doc-index`
- `scripts/dev/context.bat structure-index`
- `scripts/dev/context.bat compact-diff`
- `scripts/dev/context.bat validation-plan`
- `scripts/dev/context.bat policy-check`
- `scripts/dev/context.bat context-pack`

Issueを完了したら `scripts/dev/pull_request.bat` を使います。検証、commit、push、簡潔なPR作成を行い、PR要約を書くためだけに全diffを読む必要はありません。

## Contextを扱う規律
- 先に検索し、その後必要な箇所を読みます。
- Goal / Required / Acceptance / 作業対象が十分に分かったら、広範な探索を止めます。
- 現在のtaskのソース -> 対応test -> 直接の依存 -> 必要な詳細文書の順に確認します。
- すべての機能仕様、すべてのIssue、repository履歴、全log、全diffを既定で読み込まないでください。
- 生成されたContext Pack/index/図は案内情報として扱い、正本の代わりにしません。
- symbol/graph/diffは必要な範囲に絞って確認します。
- 無関係なrefactorや先送りした機能を現在のIssueへ混ぜません。
- 要約で不足がある場合は、元のソース/test/文書へ戻ります。
- 未実行の検証は `Unverified` と報告します。

## 検証の規律
変更の種類に合わせて根拠を選びます。まず対象を絞った検証を行い、必要な場合に完了条件まで広げます。対象が0件の成功commandは根拠になりません。UI自体がAcceptanceの対象なら画面を目視確認してください。package変更は、ソースだけでは不十分な場合、生成したartifactを検証してください。

CIでは簡潔なarchitecture policy checkerも実行します。確認済みの違反はerror、不確かな設計上の兆候はwarning/review候補として扱います。

## 情報ごとの責務
- README: 人向けの概要とsetup案内
- AI_CONTEXT: AI向けの簡潔なrouting
- 現状: 現在の機能と阻害要因
- 責務マップ: file/moduleの案内
- docs: 説明と機能設計
- specification: 規範ルール
- Issue: 要件、優先度、未完了作業
- sourceとtest: 実装済みの挙動

詳細仕様をこれらの場所へ重複して書かないでください。

## ライセンス
MIT
