# Go CLI利用ガイド

## 背景と目的

Tomiya Code AtlasはPython版からGo版へ移行しています。Go版CLIは言語別parser helperを起動し、共通IRから図を生成します。このガイドでは、Go CLIで一つのsource fileから複数の図を生成して保存する方法を説明します。

## 使い始める

Windows配布物またはGitHub Actionsが公開するGo配布artifactを展開し、`tomiya-code-atlas.exe`と`backends/`フォルダーを同じ配布folderに置いてください。`backends/manifest.json`と選択した言語のparser helperが必要です。利用者側のPython環境は不要です。

コマンドプロンプトで次のように実行します。

```bat
tomiya-code-atlas.exe analyze path\to\source.go --operation call-graph --operation class-diagram --output-dir atlas-output
```

リポジトリ内でbuildした配布物は、rootからlauncherを使って実行できます。

```bat
run_dist.bat analyze path\to\source.go --operation call-graph --operation class-diagram --output-dir atlas-output
```

`--operation`を必要な図ごとに繰り返し、`--output-dir`へ保存folderを指定してください。入力は現在一つのsource fileです。拡張子から解析言語を判定し、同じ配布folderのmanifestからparser helperを選びます。

## 結果とエラー

成功すると、入力file名とoperation名を組み合わせたMarkdown fileを一つずつ保存します。上の例では`source_call-graph.md`と`source_class-diagram.md`が`atlas-output/`に作成されます。各fileにはMermaid図が含まれます。

入力file、出力folder、operation、配布assetに問題があると、日本語のエラーと原因を表示します。`tomiya-code-atlas.exe analyze --help`は引数の説明を、`tomiya-code-atlas.exe --help`はcommand一覧を表示します。

## 現在の範囲

現在のGo CLIが対応するoperationは`call-graph`と`class-diagram`です。単一fileの解析が対象です。project folderのrecursive scan、複数fileをまとめた意味解析、その他の図やコメント生成、GUIは後続の移行Issueで対応します。
