# Tomiya Code Atlas

Tomiya Code Atlas は、ソースコードの構造・振る舞い・依存関係・責務を解析し、図表・コメント・設計評価として可視化するためのツール群です。

目的は、コードを読む前に「何があるか」「どこから呼ばれるか」「何に依存するか」「どこが複雑か」を短時間で把握できる状態を作ることです。

## 主な機能

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

リポジトリのルートで `build_exe.bat` を実行してください。`.venv` がない場合は必要なPython環境の準備も試みます。ビルド結果は `.build\dist\tomiya-code-atlas\` に作られ、`run_dist.bat` で起動できます。必要なSDKが見つからない場合は、バッチが不足項目を表示します。

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

アプリケーション全体は UPD Commander Base Design を参考に UI / Process / Data の責務を分けます。Commander は呼び出しの交通整理、Messenger は境界通信のみを担当し、実処理を持ちません。

規定は [`specification/architecture-policy.md`](specification/architecture-policy.md)、日本語での説明は [`docs/jp/構成/UPDコマンダー適用.md`](docs/jp/構成/UPDコマンダー適用.md) を参照してください。

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
app.py             # アプリケーション起動入口
```

文書は日本語を正本とし、[`docs/jp/現状.md`](docs/jp/現状.md)で現在の能力と既知制約を、[`docs/jp/責務マップ.md`](docs/jp/責務マップ.md)で責務から探す場所を確認できます。利用者向けの英語版は[`docs/en/README.md`](docs/en/README.md)から、[ビルド手順](docs/en/Building.md)、[GUIガイド](docs/en/GUI-Usage.md)、[CLIリファレンス](docs/en/CLI-Reference.md)を参照できます。

## 配布フォルダーの構成

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

## シーケンス図の設定

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

## Windowsでのセットアップと起動

開発元からクローンした場合は、次のツールをインストールします。

- Python 3.12 x64を推奨。Python 3.11以降も利用できます: [Windows向けPython配布](https://www.python.org/downloads/windows/)。`py -3.12` を優先して使い、利用できない場合は `python` コマンドから3.11以降を起動します。
- .NET SDK 10 x64: [.NET 10ダウンロード](https://dotnet.microsoft.com/download/dotnet/10.0)。
- JDK 25 x64: [Eclipse Temurin 25](https://adoptium.net/temurin/releases/?version=25&os=windows&arch=x64&package=jdk)。JDKの`bin`をPATHに追加すれば利用できます。`JAVA_HOME`を設定した場合はその値を使います。

Python Launcher (`py`) にPython 3.12がない場合は、`python` コマンドでPython 3.11以降が起動するようPATHを設定してください。リポジトリのルートで配布EXEを作るときは `build_exe.bat` を実行します。このバッチは `.venv` がなければ `scripts\build\setup.bat` を呼び出して準備します。JDKは `JAVA_HOME`、またはPATH上の `javac` から検出します。Apache MavenはPATHに見つからない場合、公式配布物を取得して`.build\tools\`内に配置し、SHA-512 checksumを検証します（初回のみネットワーク接続が必要です）。

```bat
build_exe.bat
run_dist.bat
```

`scripts\build\setup.bat` はリポジトリ内の `.venv` を作り、実行・テスト・EXE作成に必要なPythonパッケージをそこへインストールします。ソース版を起動する場合は `scripts\build\run_source.bat` を使います。Pythonや依存パッケージをグローバル環境へインストールしません。

### 成果物を作る

`scripts\build\package.bat` はPython wheelとsource archiveを `.build\packages\` に作ります。これはPython packageであり、WindowsアプリのEXEではありません。

配布用Windowsアプリのbuildには、Python 3.11以降、.NET SDK 10、JDK 25が必要です。JDKの`bin`をPATHに追加するか、`JAVA_HOME`を設定してください。Apache Mavenは`build_exe.bat`が自動で準備します。次のコマンドで前提を確認できます。

```powershell
py -3.12 --version       # 3.12がない場合は python --version（3.11以降）
dotnet --list-sdks       # 10.x SDKが表示される
javac -version           # 25.xと表示される
```


```bat
build_exe.bat
run_dist.bat
```

`build_exe.bat` はJava/C# backendとPyInstaller onedirアプリを作り、`run_dist.bat` はビルド済みの `.build\dist\tomiya-code-atlas\tomiya-code-atlas.exe` を起動します。ビルド済み配布物を別の場所へ展開した場合は、そのフォルダーの `tomiya-code-atlas.exe` を直接起動します。onedir配布ではEXE単体を移動せず、フォルダー全体を使ってください。ダウンロード済み配布物の実行時には、Python、.NET SDK、JDK、Mavenの別途インストールは不要です。

リリース前の完全検証は `scripts\build\verify.bat` です。packageとEXEのビルド、配布物CLI/GUI smoke、テスト、policy checkをまとめて実行します。上記のビルド要件を満たした環境で実行してください。

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
| `scripts\dev\context.bat policy-check` | architecture / UPD boundaryを確認 |
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

`ai-context-reducer` と `upd-commander-base-design` は設計・運用の参考元であり、実行時必須依存ではありません。

## ライセンス

ソースコードは [MIT License](LICENSE) です。アプリに含まれるアトラス君の画像素材と派生アイコンには、別途 [Tomiya Character License v1.0.1](assets/characters/atlas-kun/LICENSE.md) が適用されます。素材の出典と配布内容は [素材ライセンス案内](assets/characters/atlas-kun/README.md) を参照してください。

## 開発状況

初期実装・アーキテクチャ整備中です。
