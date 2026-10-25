# Go版コマンドライン利用ガイド

## 背景と目的

Tomiya Code Atlasは、配布時にPython環境を必要としない製品を目指してGoへ移行しています。Go版CLIは、入力されたソースを言語ごとの解析補助プログラムへ渡し、共通中間表現から図を生成します。このガイドでは、ソースファイルまたはプロジェクトフォルダーを解析し、複数の図を保存する方法を説明します。

## 配布物を用意する

GitHub ActionsのWindows配布artifactを展開し、`tomiya-code-atlas-cli.exe`と`backends/`フォルダーを同じ場所に置いてください。`backends/manifest.json`と、解析する言語に対応する解析補助プログラムも必要です。利用者のPCにPythonをインストールする必要はありません。

GUIを使う場合は、同じ配布物に含まれる`tomiya-code-atlas.exe`を起動してください。操作手順は[GUI操作ガイド](GUI操作ガイド.md)を参照してください。

## ソースを解析する

コマンドプロンプトで、解析対象と作成する図、結果の保存先を指定してください。

```bat
tomiya-code-atlas-cli.exe analyze path\to\project --operation call-graph --operation class-diagram --output-dir atlas-output
```

リポジトリを使っている場合は、次のように実行できます。

```bat
run_dist.bat analyze path\to\source.go --operation call-graph --operation class-diagram --output-dir atlas-output
```

`--operation`には、作成したい図ごとに`call-graph`または`class-diagram`を指定してください。入力には一つのソースファイルまたはプロジェクトフォルダーを指定します。フォルダーを指定すると、配下から対応するソースファイルを再帰的に探し、拡張子に合った解析補助プログラムで一つずつ解析します。

各ファイルの解析結果をまとめて図を作りますが、ファイル間の名前・型の解決や、パッケージ単位の意味解析はまだ行いません。

## 結果とエラーを確認する

成功すると、入力名と図の種類を含むMarkdownファイルを保存します。上の例では`atlas-output/`に`project_call-graph.md`と`project_class-diagram.md`が作成され、各ファイルにMermaid形式の図が入ります。

入力、保存先、図の指定、配布ファイルに問題がある場合は、日本語で原因を表示します。`tomiya-code-atlas-cli.exe analyze --help`で解析引数を、`tomiya-code-atlas-cli.exe --help`でコマンド一覧を確認してください。

解析が制限時間を超えた場合は入力ファイルを確認し、解析補助プログラムとの通信に失敗した場合は配布物を再取得してください。`詳細:`以降には、原因調査に必要な技術情報を表示します。

## 現在の対応範囲

Go版GUIとCLIが作成できる図は、呼び出し図（`call-graph`）とクラス図（`class-diagram`）です。ファイルをまたいだ名前・型の解決、その他の図、コメント生成は後続の移行Issueで対応します。
