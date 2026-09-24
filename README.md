# Tomiya Code Atlas

Tomiya Code Atlas は、ソースコードの構造・振る舞い・依存関係・責務を解析し、図表・コメント・設計評価として可視化するためのツール群です。

目的は、コードを読む前に「何があるか」「どこから呼ばれるか」「何に依存するか」「どこが複雑か」を短時間で把握できる状態を作ることです。

## Main capabilities

主な対象:
- コメント生成
- クラス / オブジェクト / シーケンス / コミュニケーション図
- 状態遷移 / アクティビティ / タイミング / ユースケース図
- パッケージ / コンポーネント / デプロイメント図
- コールグラフ
- クラス責務表
- CI 解析
- 設計評価

図表の標準出力は Mermaid とし、PlantUML 等は Renderer の差し替えで追加できる構造を目指します。コメント生成は基本的に元ソースへ追記します。

## Analysis targets

主要な解析対象言語:
- Python
- GDScript
- C#
- C++
- Java
- Go

最初の解析実装対象は Python です。言語固有処理は境界へ閉じ込め、中央の解析・生成・評価処理は可能な限り言語非依存にします。

## Core architecture

```text
Source
  -> Language Adapter / Parser
  -> Common IR / shared models
  -> Analyzer / Generator / Evaluator
  -> Logical Output
  -> Renderer
  -> Mermaid / PlantUML / text / table
```

アプリケーション全体は UPD Commander Base Design を参考に UI / Process / Data の責務を分けます。Commander は呼び出しの交通整理、Messenger は境界通信のみを担当し、実処理を持ちません。

規定は [`specification/architecture-policy.md`](specification/architecture-policy.md)、説明は [`docs/architecture/upd_commander.md`](docs/architecture/upd_commander.md) を参照してください。

## Repository structure

```text
Src/
  analyzers/       # deterministic relationship / graph analysis
  models/          # shared passive data contracts
  generators/      # logical output generation
  renderers/       # Mermaid / text / other formatting
  evaluators/      # design / code-quality evaluation
  languages/       # language-specific adapters
config/            # runtime configuration and examples
tests/             # automated evidence
tools/             # Issue / PR / context helpers
docs/              # explanations, current state, routing, feature specs
specification/     # normative project rules
app.py             # application entry point
```

現在の能力・既知制約は [`docs/current_state.md`](docs/current_state.md)、責務からファイルを探す場合は [`docs/responsibility_map.md`](docs/responsibility_map.md) を参照してください。

## Runtime layout

Windows配布物は PyInstaller `onedir` を使用します。設定や今後の外部リソースをEXE本体へ埋め込まず、配布ディレクトリ内で分離します。

```text
tomiya-code-atlas/
  tomiya-code-atlas.exe
  config/
    tomiya-code-atlas.json
    tomiya-code-atlas.example.json
  _internal/
```

実行時設定は `config/tomiya-code-atlas.json` を読みます。PyInstaller版ではEXEのあるディレクトリを基準にし、ソース実行時もリポジトリの `config/` を基準にします。

## Sequence diagram settings

`config/tomiya-code-atlas.json` を編集すると起動時に読み込みます。設定例は `config/tomiya-code-atlas.example.json` を参照してください。

```json
{
  "generator_options": {
    "sequence_diagram": {
      "show_duplicate_calls": true,
      "show_returns": false
    }
  }
}
```

- `show_duplicate_calls`: 同一callerから同一calleeへの同一呼び出しを複数回表示するか
- `show_returns`: 戻り値メッセージを表示するか
- 循環呼び出しは設定にかかわらずシーケンス図から除外し、無限展開を防止します
- GUI上のチェック項目で、その実行時だけ設定を上書きできます

## Project operations

GitHub Issue をタスク台帳として扱い、原則 `1 Issue ~= 1 PR` です。

作業開始の推奨入口:

```text
prepare_work.bat
```

Linux/macOS:

```text
./prepare_work.sh
```

これは次を行います。

```text
priority-first Issue selection
  -> Task Capsule
  -> Remote Delta First
  -> Context Pack
```

個別コマンド:
- `next_issue.bat` — 最優先の actionable Issue を1件だけTask Capsule化
- `context.bat profile` — repo規模 / context budget / hotspot候補
- `context.bat doc-index` — docsの見出し索引
- `context.bat remote-delta` — ahead/behind / remote commits / changed files / bounded diff
- `context.bat compact-diff` — changed files / shortstat / commit summary
- `context.bat structure-index` — Python symbol/import index
- `context.bat validation-plan` — changed filesから検証をルーティング
- `context.bat policy-check` — architecture / UPD boundary のcompact check
- `context.bat context-pack` — 一時作業Context Pack生成
- `pull_request.bat` — validation / commit / compact summary / push / PR

Linux/macOSでは `./context.sh <command>` を使用できます。

詳細は [`docs/project_operations.md`](docs/project_operations.md) を参照してください。

## AI context policy

- Search first, read second
- Goal / Required / Acceptance / working set が揃ったら探索を止める
- Responsibility Map / structure index から対象を絞る
- remote更新はfull rereadよりcompact deltaを先に見る
- full diff / full logs / all docs / all Issues を通常コンテキストへ入れない
- generated Context Pack / index は原典の代替にしない
- unrelated refactor を混ぜない
- 検証できなかった範囲は `Unverified` とする
- 正確性をコンテキスト削減量より優先する

AI向け入口は `AI_CONTEXT.md` と `AGENTS.md` です。

## Specifications

README は概要だけを保持します。個別機能の要件・Acceptance Criteria は `docs/specs/` と GitHub Issues、横断的な必須規則は `specification/` を Source of Truth とします。

`ai-context-reducer` と `upd-commander-base-design` は設計・運用の参考元であり、実行時必須依存ではありません。

## License

MIT License

## Status

初期実装・アーキテクチャ整備中です。
