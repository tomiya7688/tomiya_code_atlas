# Tomiya Code Atlas

Tomiya Code Atlas は、ソースコードの構造・振る舞い・依存関係・責務を解析し、図表・コメント・設計評価として可視化するためのツール群です。

目的は、コードを読む前に「何があるか」「どこから呼ばれるか」「何に依存するか」「どこが複雑か」を短時間で把握できる状態を作ることです。

製品本体はGoへ移行中です。現在のGo CLIはhelp/versionのみのbootstrapで、解析機能やGUIは未移植です。Go配布物にはCPython ASTとRoslyn C# parser helperを含みますが、CLIからの起動・処理連携は後続Issueで実装します。既存機能の説明は移行中のPython source版を指し、移行完了までは [Issue #27](https://github.com/tomiya7688/tomiya_code_atlas/issues/27) と子Issueで進捗を管理します。

## 主な機能

主な対象:
- コメント生成
- クラス / オブジェクト / シーケンス / コミュニケーション図
- 状態遷移 / アクティビティ / タイミング / ユースケース図
- パッケージ / コンポーネント / デプロイメント図
- コールグラフ
- [静的仕様書](docs/jp/機能仕様/静的仕様書.md)（Pythonファイル／プロジェクト）
- 静的仕様書（Python単一ファイル）
- クラス責務表
- CI 解析
- 設計評価

図表の標準出力は Mermaid とし、PlantUML 等は Renderer の差し替えで追加できる構造を目指します。コメント生成は基本的に元ソースへ追記します。

## 主な解析対象

主要な解析対象言語:
- Python
- GDScript
- C#
- C++
- Java
- Go

最初の解析実装対象は Python です。言語固有処理は境界へ閉じ込め、中央の解析・生成・評価処理は可能な限り言語非依存にします。

## Windows配布アプリをビルドする

Go版への移行を開始しています。ルートで `build_exe.bat` を実行すると、Go CLIとCPython/Roslyn parser helperを含むone-dir配布物を作成します。現段階のGo CLIはhelp/versionのみの移行用bootstrapです。GitHub Actionsの `Go EXE` workflowからWindows配布artifactを取得できます。

Python版は移行期間中の参照用ソースとして残します。Python parser helperはCPython標準ASTで文法を解析し、配布時はPyInstaller one-dir内へ対応するCPython runtimeを同梱します。アーキテクチャ規則として、全言語の構造解析backendで対象言語の実文法parserまたはcompiler ASTを使います。

## 基本アーキテクチャ

```text
Source
  -> Language Adapter / Parser
  -> Common IR / shared models
  -> Analyzer / Generator / Evaluator
  -> Logical Output
  -> Renderer
  -> Mermaid / PlantUML / text / table
```

アプリケーション全体では UI / Process / Data の責務を分け、処理の振り分けと境界通信を薄く保ちます。解析や入出力などの実処理は、それぞれの専門moduleが担当します。

規定は [`specification/architecture-policy.md`](specification/architecture-policy.md)、moduleごとの責務は [`docs/jp/責務マップ.md`](docs/jp/責務マップ.md) を参照してください。

## repository構成

```text
Src/
  analyzers/       # 関係・graphの決定的な解析
  models/          # 受動的な共通data契約
  generators/      # 論理出力の生成
  renderers/       # Mermaid / textなどの出力形式
  evaluators/      # 設計・コード品質の評価
  languages/       # 言語固有のadapter
config/            # 実行時設定と例
tests/             # 自動検証
tools/             # Issue / PR / context補助
scripts/dev/       # 開発用Issue / context / PR入口
docs/jp/           # 日本語正本
docs/en/           # 日本語正本から作成した英語版
specification/     # 規範となるプロジェクトルール
go/                # Go版のmoduleと製品コマンド
app.py             # 移行中のPython版アプリケーション入口
```

文書は日本語を正本とし、[`docs/jp/現状.md`](docs/jp/現状.md)で現在の能力と既知制約を、[`docs/jp/責務マップ.md`](docs/jp/責務マップ.md)で責務から探す場所を確認できます。利用者向けの英語版は[`docs/en/README.md`](docs/en/README.md)から、[ビルド手順](docs/en/Building.md)、[GUIガイド](docs/en/GUI-Usage.md)、[CLIリファレンス](docs/en/CLI-Reference.md)を参照できます。

## Windowsでのセットアップと起動

開発元からクローンした場合は、次のツールをインストールします。

- Go 1.22以降: [GoのWindows配布](https://go.dev/dl/)。Go版EXEをローカルbuildするときに使います。
- .NET 10 SDK: [Microsoft .NETダウンロード](https://dotnet.microsoft.com/download/dotnet/10.0)。C# Roslyn helperのself-contained buildに使います。配布物の利用者は.NETを別途用意する必要がありません。
- Python 3.11以降: [Windows向けPython配布](https://www.python.org/downloads/windows/)。source実行、test、parser helperのbuildに使います。配布物はCPython runtimeを同梱します。

Go版EXEとparser helperのbuildにはGo 1.22以降、Python 3.11以降、.NET 10 SDKと `scripts\build\setup.bat` が必要です。配布物の利用者はGo、Python、.NETを別途用意する必要がありません。

`scripts\build\setup.bat` はリポジトリ内の `.venv` を作り、source testとPyInstaller helper buildに必要なpackageをインストールします。ソース版を起動する場合は `scripts\build\run_source.bat` を使います。Pythonや依存packageをglobal環境へインストールしません。

```bat
build_exe.bat
run_dist.bat
```

`build_exe.bat` はGo CLIとCPython AST helper、.NET runtimeを含むRoslyn C# helperを作り、`run_dist.bat` はGo CLIを起動します。現時点のGo版CLIは移行用bootstrapで、解析機能は後続Issueで順次移植します。GitHub Actionsの `Go EXE` workflowはWindows配布物を作成・起動確認し、artifactとして公開します。

`scripts\build\verify.bat` はGo EXEとPython/C# parser helperのbuild・起動smoke、Python source test、policy checkを実行します。v1.0.0のrelease workflowはGo移行が完了するまで用意しません。

## 開発用コマンド

GitHub Issueをタスク台帳として扱い、原則 `1 Issue ~= 1 PR` です。開発補助コマンドはルートから `scripts\dev\` へまとめています。

```bat
scripts\dev\prepare_work.bat
```

Linux/macOSでは `./scripts/dev/prepare_work.sh` を使います。これは優先Issueの選択、Task Capsule作成、remote delta確認、Context Pack生成を行います。

| コマンド | 用途 |
| --- | --- |
| `scripts\dev\next_issue.bat` | 最優先の actionable Issue をTask Capsule化 |
| `scripts\dev\context.bat profile` | repo規模とcontext使用量を確認 |
| `scripts\dev\context.bat remote-delta` | ahead/behindとリモート変更を確認 |
| `scripts\dev\context.bat validation-plan` | 変更ファイルから検証を選ぶ |
| `scripts\dev\context.bat policy-check` | architecture boundaryを確認 |
| `scripts\dev\context.bat context-pack` | 一時Context Packを生成 |
| `scripts\dev\reducer.bat setup` | 開発専用reducerを準備 |
| `scripts\dev\pull_request.bat` | 検証、commit、push、PR作成 |

他のcontextコマンドは `scripts\dev\context.bat --help` を参照してください。Linux/macOSでは同じ場所の `context.sh`、`reducer.sh`、`prepare_work.sh` を使います。詳細は [`docs/jp/開発運用.md`](docs/jp/開発運用.md) を参照してください。

## AI向け作業方針

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

## 仕様と規範文書

READMEは概要だけを保持します。日本語の個別機能仕様は `docs/jp/機能仕様/`、Issueごとの要件・Acceptance CriteriaはGitHub Issues、横断的な必須規則は `specification/` を正本とします。`docs/en/` は日本語正本から作る英語版です。

`ai-context-reducer` は開発専用の補助toolで、実行時必須依存ではありません。

## ライセンス

ソースコードは [MIT License](LICENSE) です。アプリに含まれるアトラス君の画像素材と派生アイコンには、別途 [Tomiya Character License v1.0.1](assets/characters/atlas-kun/LICENSE.md) が適用されます。素材の出典と配布内容は [素材ライセンス案内](assets/characters/atlas-kun/README.md) を参照してください。

## 開発状況

初期実装・アーキテクチャ整備中です。
