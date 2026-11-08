# Tomiya Code Atlas

Tomiya Code Atlas は、ソースコードの構造・振る舞い・依存関係・責務を解析し、図表・コメント・設計評価として可視化するためのツール群です。

目的は、コードを読む前に「何があるか」「どこから呼ばれるか」「何に依存するか」「どこが複雑か」を短時間で把握できる状態を作ることです。

製品本体はGoへ移行中です。Go CLIには16個のoperation IDを登録し、Windows GUIは入力形式・言語parser・workflowや配置設定に応じて実行できる成果物を選べるようにします。オブジェクト図、状態図、タイミングチャート、ユースケース図は現時点でPython sourceを含む入力が対象です。project folder内のsource fileは個別に解析するため、file間の名前・型解決はまだ行いません。コメント生成では元のソースを変更しません。実行方法は[CLI利用ガイド](docs/jp/CLI%E5%88%A9%E7%94%A8%E3%82%AC%E3%82%A4%E3%83%89.md)、GUI操作は[GUI操作ガイド](docs/jp/GUI%E6%93%8D%E4%BD%9C%E3%82%AC%E3%82%A4%E3%83%89.md)を参照してください。Goへの移行状況は [Issue #27](https://github.com/tomiya7688/tomiya_code_atlas/issues/27) と子Issueで管理します。

## 主な機能

対応する成果物と入力条件は、[CLI利用ガイド](docs/jp/CLI%E5%88%A9%E7%94%A8%E3%82%AC%E3%82%A4%E3%83%89.md)と[GUI操作ガイド](docs/jp/GUI%E6%93%8D%E4%BD%9C%E3%82%AC%E3%82%A4%E3%83%89.md)で確認できます。Go版のコメント生成は元ソースを変更せず、コメント付きコピーを出力します。

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

ルートで`build_exe.bat`を実行すると、GUIとCLIを同じ`tomiya-code-atlas.exe`で提供し、各言語のparser helperと必要なruntimeを含むone-dir配布物を作成します。GitHub Actionsの`Go EXE` workflowからWindows配布artifactを取得できます。

Python版は移行期間中の参照用ソースとして残します。Python parser helperはCPython標準ASTで文法を解析し、配布時はPyInstaller one-dir内へ対応するCPython runtimeを同梱します。アーキテクチャ規則として、全言語の構造解析backendで対象言語の実文法parserまたはcompiler ASTを使います。

### Windows配布artifactの取得と検証

PRの動作確認では、対象pull requestの成功した`Go EXE` workflow runを開き、`tomiya-code-atlas-windows-x64-pr-<番号>` artifactをダウンロードしてください。この検証用ZIPとchecksumは7日間保持し、provenance attestationはありません。正式配布artifactは、正規repositoryへのpushまたはworkflow手動実行が成功した後に`tomiya-code-atlas-go-windows-x64`として保存します。forkのpull requestではartifactを公開しません。

```powershell
gh attestation verify tomiya-code-atlas-windows-x64.zip -R tomiya7688/tomiya_code_atlas --signer-workflow tomiya7688/tomiya_code_atlas/.github/workflows/go-exe.yml --source-ref <source-ref> --source-digest <commit-sha>
```

`<source-ref>`と`<commit-sha>`はworkflow runの表示値に置き換えてください。source refはbranch pushなら`refs/heads/<branch>`、tag pushなら`refs/tags/<tag>`です。ZIP内の`SHA256SUMS.txt`で、展開したparser helper等のSHA-256を照合してください。この検証手順は本workflowが作ったartifactを対象とし、過去の成果物にはそのprovenanceがありません。provenance attestationはWindows Authenticode署名を付与しません。正式releaseのAuthenticode署名は、信頼された証明書を用意するまで未対応です。

workflowは配布ZIP、両方のSHA-256 manifest、および各同梱binaryを個別にattestします。ZIPの検証と同じ`gh attestation verify` optionsを使い、対象pathをartifact内の `SHA256SUMS.txt` または展開した各binaryへ置き換えて検証してください。各binaryのhashもmanifestと照合してください。

GitHub CLIは既定でattestationを最大30件取得します（[公式manual](https://cli.github.com/manual/gh_attestation_verify)）。対象runのattestationがその範囲外で見つからない場合は、source refとcommit SHAの検証を維持し、対象runを含む件数を`--limit`に指定して再検証してください（例: `--limit 100`）。

現行workflowが生成する成果物とv1.0.0のrelease gateは[リリース確認表](docs/jp/%E3%83%AA%E3%83%AA%E3%83%BC%E3%82%B9%E7%A2%BA%E8%AA%8D%E8%A1%A8.md)を参照してください。

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

規定は [`specification/architecture-policy.md`](specification/architecture-policy.md)、moduleごとの責務は [`docs/jp/責務マップ.md`](docs/jp/%E8%B2%AC%E5%8B%99%E3%83%9E%E3%83%83%E3%83%97.md) を参照してください。

## repository構成

```text
.github/
  workflows/                 # GitHub ActionsのCI／配布workflow
  config/
    oop-design-checker.json  # GitHub CI専用の静的解析設定
assets/                      # アプリケーションの静的素材
config/                      # 実行時設定と例
docs/
  jp/                        # 日本語正本
  en/                        # 日本語正本から作成した英語版
go/                          # Go製品本体とCLI
parser-backends/             # 言語別parser source、manifest、license
scripts/
  build/                     # build、source起動、検証の内部処理
  dev/                       # Issue、context、PRの開発補助
specification/               # 規範となるプロジェクトルール
Src/
  app.py                     # Python参照実装のmodule entry
  analyzers/                 # 関係・graphの決定的な解析
  models/                    # 受動的な共通data契約
  generators/                # 論理出力の生成
  renderers/                 # Mermaid / textなどの出力形式
  evaluators/                # 設計・コード品質の評価
  languages/                 # 言語固有のadapter
tests/                       # 自動検証
tools/                       # Issue / PR / context補助
```

ルート直下には、利用者向けのWindows build/run入口、リポジトリ案内、Python参照実装用metadataを置きます。各ファイルの役割は次のとおりです。

| ルート直下のファイル                         | 役割                                                                                                               |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `README.md`                                  | 人向けの概要、セットアップ、利用入口を説明します。                                                                 |
| `LICENSE`                                    | MITライセンスを記載します。                                                                                        |
| `AGENTS.md`                                  | エージェントが作業時に守る安定した規則を記載します。                                                               |
| `AI_CONTEXT.md`                              | AIが正本や責務の場所を探すための短い案内です。                                                                     |
| `.gitignore`                                 | build成果物、cache、ローカル状態をGitの追跡対象から除外します。                                                    |
| `.updcommanderignore`                        | UPD Commanderの解析対象から除外するpathを指定します。                                                              |
| `build_exe.bat`, `run_dist.bat`              | Windows配布物のbuildと起動に使う2つの直接入口です。内部処理は `scripts/build/` にあります。                        |
| `pyproject.toml`, `setup.cfg`, `MANIFEST.in` | Python参照実装のeditable install・testとparser helper build用の開発metadataです。Go製品の配布buildには使いません。 |

Python参照実装は `python -m Src.app` または `scripts\build\run_source.bat` から起動します。Windows配布物の利用者は `build_exe.bat` と `run_dist.bat` を使ってください。

文書は日本語を正本とし、[`docs/jp/現状.md`](docs/jp/%E7%8F%BE%E7%8A%B6.md)で現在の能力と既知制約を、[`docs/jp/責務マップ.md`](docs/jp/%E8%B2%AC%E5%8B%99%E3%83%9E%E3%83%83%E3%83%97.md)で責務から探す場所を確認できます。利用者向けの英語版は[`docs/en/README.md`](docs/en/README.md)から、[ビルド手順](docs/en/Building.md)、[GUIガイド](docs/en/GUI-Usage.md)、[CLIリファレンス](docs/en/CLI-Reference.md)を参照できます。

## Windowsでのセットアップと起動

開発元からクローンした場合は、次のツールをインストールします。

- Go 1.27.x: [GoのWindows配布](https://go.dev/dl/)。Go版EXEとGo parser helperのbuildに使います。
- .NET 10 SDK: [Microsoft .NETダウンロード](https://dotnet.microsoft.com/download/dotnet/10.0)。C# Roslyn helperのself-contained buildに使います。配布物の利用者は.NETを別途用意する必要がありません。
- JDK 25: Java parser helperのbuildとprivate runtime作成に使います。配布物の利用者はJavaを別途用意する必要がありません。
- Apache Maven: [Mavenの配布案内](https://maven.apache.org/download.cgi)。JavaParser helperのbuildに使います。利用者向けartifactにはMavenを含めません。
- Python 3.11.18、3.12.16、3.13.18、3.14.10、3.15.2以降: [Windows向けPython配布](https://www.python.org/downloads/windows/)。source実行、test、parser helperのbuildに使います。対応する安定版patch floorは[開発運用](docs/jp/%E9%96%8B%E7%99%BA%E9%81%8B%E7%94%A8.md#windows%E3%83%93%E3%83%AB%E3%83%89%E3%81%AE%E6%A4%9C%E8%A8%BC)を参照してください。配布物はCPython runtimeを同梱します。

Go版EXEと全parser helperのbuildにはGo 1.27.x、上記のsecurity floorを満たすPython、.NET 10 SDK、JDK 25、Apache Mavenと `scripts\build\setup.bat` が必要です。配布物の利用者はこれらの開発toolchainを別途用意する必要がありません。

`scripts\build\setup.bat` はリポジトリ内の `.venv` を作り、source testとPyInstaller helper buildに必要なpackageをインストールします。ソース版を起動する場合は `scripts\build\run_source.bat` を使います。Pythonや依存packageをglobal環境へインストールしません。

```bat
build_exe.bat
run_dist.bat
```

`build_exe.bat` は`tomiya-code-atlas.exe`（引数なしでGUI、CLI command指定でconsole CLI）とCPython/GDScript、Roslyn C#、Clang C++、JavaParser helperを作り、`run_dist.bat` はGUIまたはconsole CLIを起動します。Java helperはjlinkで作成したprivate runtimeを同梱します。GitHub Actionsの`Go EXE` workflowは配布物をbuildし、helperと解析結果を確認してからartifactとして公開します。

`scripts\build\verify.bat` はGo EXEと全parser helperのbuild・起動smoke、Python source test、policy checkを実行します。v1.0.0のrelease workflowはGo移行が完了するまで用意しません。

## 開発用コマンド

GitHub Issueをタスク台帳として扱い、原則 `1 Issue ~= 1 PR` です。開発補助コマンドはルートから `scripts\dev\` へまとめています。

```bat
scripts\dev\prepare_work.bat
```

Linux/macOSでは `./scripts/dev/prepare_work.sh` を使います。これは優先Issueの選択、Task Capsule作成、remote delta確認、Context Pack生成を行います。

| コマンド                                  | 用途                                       |
| ----------------------------------------- | ------------------------------------------ |
| `scripts\dev\next_issue.bat`              | 最優先の actionable Issue をTask Capsule化 |
| `scripts\dev\context.bat profile`         | repo規模とcontext使用量を確認              |
| `scripts\dev\context.bat remote-delta`    | ahead/behindとリモート変更を確認           |
| `scripts\dev\context.bat validation-plan` | 変更ファイルから検証を選ぶ                 |
| `scripts\dev\context.bat policy-check`    | architecture boundaryを確認                |
| `scripts\dev\context.bat context-pack`    | 一時Context Packを生成                     |
| `scripts\dev\reducer.bat setup`           | 開発専用reducerを準備                      |
| `scripts\dev\pull_request.bat`            | 検証、commit、push、PR作成                 |

他のcontextコマンドは `scripts\dev\context.bat --help` を参照してください。Linux/macOSでは同じ場所の `context.sh`、`reducer.sh`、`prepare_work.sh` を使います。詳細は [`docs/jp/開発運用.md`](docs/jp/%E9%96%8B%E7%99%BA%E9%81%8B%E7%94%A8.md) を参照してください。

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
