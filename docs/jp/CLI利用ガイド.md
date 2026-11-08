# Go CLI利用ガイド

## 背景と目的

Tomiya Code AtlasはPython版からGo版へ移行しています。Go版CLIは言語別parser helperを起動し、共通IRから図、コメント付きsource copy、静的仕様書、設計品質レポートを生成します。このガイドでは、Go CLIでsource fileまたはproject folderを解析して複数の結果を保存する方法を説明します。

## 使い始める

Windows配布物またはGitHub Actionsが公開するGo配布artifactを展開し、`tomiya-code-atlas.exe`と`backends/`フォルダーを同じ配布folderに置いてください。`backends/manifest.json`と選択した言語のparser helperが必要です。利用者側のPython環境は不要です。

コマンドプロンプトで次のように実行します。

```bat
tomiya-code-atlas.exe analyze path\to\project --operation call-graph --operation class-diagram --operation ci-graph --operation design-quality --output-dir atlas-output
```

リポジトリ内でbuildした配布物は、rootからlauncherを使って実行できます。

```bat
run_dist.bat analyze path\to\project --operation call-graph --operation class-diagram --operation ci-graph --operation design-quality --output-dir atlas-output
```

`--operation`を必要な結果ごとに繰り返し、`--output-dir`へ保存folderを指定してください。入力には一つのsource fileまたはproject folderを指定できます。folderを指定すると対応するsourceを下位folderまで再帰的に探索し、各fileを拡張子に対応するmanifest上のparser helperで個別に解析してCommon IRを統合します。複数file間の型解決やpackage単位の意味解析はまだ行いません。通常のソース解析ではYAML設定を対象外にします。YAMLだけを指定すると、対応ソースがないことを案内します。デプロイメント設定の内部探索ではYAMLを引き続き扱います。

folder入力で`comments`を選ぶ場合、`--output-dir/comments`は入力folderと重ならないpathを指定してください。両者が同じpath、親子関係になるpathは指定できません。CLIはrecursive scanの前に重複を拒否し、入力内に作ったcomment copyの再検出と入出力folderの重なりを防ぎます。入力folderと別の出力先（例: `atlas-output`）を指定してください。

## 結果とエラー

図、静的仕様書、設計品質レポートは`atlas-output/`直下へMarkdown fileとして保存します。上の例では`project_call-graph.md`、`project_class-diagram.md`、`project_static-spec.md`、`project_design-quality.md`を作成します。`comments`は言語のcomment記法を使って候補をsourceへ挿入し、`atlas-output/comments/`へ保存します。folder入力では元の相対folder構成を保ち、単一file入力では`<file名>_commented.<拡張子>`として保存します。元のsource fileは変更しません。

入力file、出力folder、operation、配布assetに問題があると、日本語のエラーと原因を表示します。`tomiya-code-atlas.exe analyze --help`は引数の説明を、`tomiya-code-atlas.exe --help`はcommand一覧を表示します。

parserが失敗した場合も、原因と次の確認手順を日本語で案内します。たとえば、制限時間を超えた場合は入力fileを確認し、parserとの通信形式が不正な場合は配布物を再取得してください。`詳細:`にはparserが返した技術情報を表示します。

## 現在の範囲

現在のGo CLIは16個のoperation IDを登録しています。実行できるoperationは入力形式、言語parser、workflowや配置設定の有無で変わります。オブジェクト図・状態図・タイミングチャート・ユースケース図は現時点でPython sourceを含む入力が対象です。パッケージ図・コンポーネント図はsourceを含むproject folder、配置図は配置設定を含むproject folder、CI図は単一workflow YAML fileまたはproject folder内の`.github/workflows/`にあるYAML fileで利用できます。CLIとGUIは対応しないoperationをエラーまたは選択不可として案内します。各source fileは独立して解析するため、fileをまたぐ名前・型解決はまだ行いません。コメント付きcopyは元sourceを変更しません。operation IDと説明は`tomiya-code-atlas.exe analyze --help`で確認してください。操作方法は[GUI操作ガイド](GUI%E6%93%8D%E4%BD%9C%E3%82%AC%E3%82%A4%E3%83%89.md)を参照してください。
