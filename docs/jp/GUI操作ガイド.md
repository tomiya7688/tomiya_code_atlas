# Go版GUIの使い方

英語版: [GUI Usage](../en/GUI-Usage.md)

Windows版 `tomiya-code-atlas.exe` を引数なしで起動するとGUIを開きます。Go版GUIでは、ファイルまたはプロジェクトフォルダーを解析し、図、レポート、コメント付きコピーを生成できます。複数の成果物を同時に選び、保存先フォルダーを指定できます。CLIの `analyze` コマンドも引き続き使えます。

## 解析結果を生成する

1. 「ファイル」または「プロジェクトフォルダー」を選び、「ファイルを選ぶ」または「フォルダーを選ぶ」を押してください。ショートカットは `Alt+O` です。
1. 入力に応じて有効になった成果物を一つ以上選んでください。選択できる成果物は、入力形式、解析対象の言語、workflowや配置設定の有無で変わります。
1. 「保存先を選ぶ」を押して結果の保存フォルダーを指定してください。ショートカットは `Alt+P` です。
1. 「解析して結果を作る」を押してください。ショートカットは `Alt+R` です。
1. 完了後、画面に表示された生成ファイルのpathを確認してください。

解析中のエラーは画面に表示されます。入力fileと出力folderはネイティブdialogで選択します。解析はGoアプリと同梱parser helperがローカルで実行し、選択したソースを外部サービスへ送信しません。

## 動作環境と制約

Windows 10/11を対象にしたGUIです。WailsはWindows上でWebView2 Runtimeを使います。Windows 11および多くのWindows 10環境には含まれていますが、未導入の環境ではMicrosoft Edge WebView2 Runtimeをインストールしてください。GUIにPython/Tkの実行環境は必要ありません。言語別解析helperは配布物に同梱されています。

Go版GUIには16種類の成果物があります。単一の対応source fileでは基本8種類を選べます。Python fileではPython source向けの4種類が加わり、合計12種類を選べます。project folderではパッケージ図・コンポーネント図が加わり、workflowと配置設定がある場合はCI図・配置図も選べます。単一YAML fileではCI図を選べます。灰色の項目は選んだ入力では使えません。オブジェクト図・状態図・タイミングチャート・ユースケース図は現時点でPython sourceを含む入力に限ります。図とレポートは選んだ保存先フォルダー直下にMarkdown fileとして保存し、コメント付きcopyはその中の`comments/`へ元のfolder構成を保って保存します。元のsource fileは変更しません。入力がproject folderの場合、`<保存先>/comments/`が入力folderと同じ場所、その親、またはその子にならないよう保存先を選んでください。重なる場合は解析前にエラーで拒否します。コメント付きcopyのGUI preview編集は未対応です。配置図では、Dockerfile/Composeから読み取った設定とsource importから推定した関係を区別して統合します。配布EXEは [Go EXE workflow](https://github.com/tomiya7688/tomiya_code_atlas/actions/workflows/go-exe.yml) の成功runから取得してください。

## CLI

```text
tomiya-code-atlas.exe analyze sample.py --operation call-graph --operation class-diagram --output-dir atlas-output
```

詳細は[CLI利用ガイド](CLI%E5%88%A9%E7%94%A8%E3%82%AC%E3%82%A4%E3%83%89.md)を参照してください。

## 参照用のPython版

移行中のPythonソース版GUIを使う開発者は、環境を準備してから `python -m Src.app gui` を実行してください。これはGo配布物とは別の参照実装です。

## 開発時のCSS整形

Node.js 24を用意し、リポジトリのルートで `npm ci --prefix tools/web-format --ignore-scripts` を実行して、固定版のPrettierを準備してください。`npm --prefix tools/web-format run format` でGUIのCSSを整形し、`npm --prefix tools/web-format run check` で整形状態を確認できます。
