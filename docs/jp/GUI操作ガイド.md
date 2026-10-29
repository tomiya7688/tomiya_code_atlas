# Go版GUIの使い方

英語版: [GUI Usage](../en/GUI-Usage.md)

Windows版 `tomiya-code-atlas.exe` を引数なしで起動するとGUIを開きます。Go版GUIはファイル／プロジェクトフォルダーから呼び出し図とクラス図を生成します。複数の図を同時に選び、保存先フォルダーを指定できます。CLIの `analyze` コマンドも引き続き使えます。

## 図を生成する

1. 「ファイル」または「プロジェクトフォルダー」を選び、「ファイルを選ぶ」または「フォルダーを選ぶ」を押してください。ショートカットは `Alt+O` です。
2. 「呼び出し図」「クラス図」から必要な図を一つ以上選んでください。
3. 「保存先を選ぶ」を押して結果の保存フォルダーを指定してください。ショートカットは `Alt+P` です。
4. 「解析して図を作る」を押してください。ショートカットは `Alt+R` です。
5. 完了後、画面に表示されたMarkdownファイルのpathを確認してください。

解析中のエラーは画面に表示されます。入力fileと出力folderはネイティブdialogで選択します。解析はGoアプリと同梱parser helperがローカルで実行し、選択したソースを外部サービスへ送信しません。

## 動作環境と制約

Windows 10/11を対象にしたGUIです。WailsはWindows上でWebView2 Runtimeを使います。Windows 11および多くのWindows 10環境には含まれていますが、未導入の環境ではMicrosoft Edge WebView2 Runtimeをインストールしてください。GUIにPython/Tkの実行環境は必要ありません。言語別解析helperは配布物に同梱されています。

現行GUIで選べる図はGo配布版の `call-graph` と `class-diagram` です。コメント生成、プレビュー編集、その他の図表は後続Issueで移植します。配布EXEは [Go EXE workflow](https://github.com/tomiya7688/tomiya_code_atlas/actions/workflows/go-exe.yml) の成功runから取得してください。

## CLI

```text
tomiya-code-atlas.exe analyze sample.py --operation call-graph --operation class-diagram --output-dir atlas-output
```

詳細は[CLI利用ガイド](CLI利用ガイド.md)を参照してください。

## 参照用のPython版

移行中のPythonソース版GUIを使う開発者は、環境を準備してから `python -m Src.app gui` を実行してください。これはGo配布物とは別の参照実装です。

## 開発時のCSS整形

Node.js 24を用意し、リポジトリのルートで `npm ci --prefix tools/web-format --ignore-scripts` を実行して、固定版のPrettierを準備してください。`npm --prefix tools/web-format run format` でGUIのCSSを整形し、`npm --prefix tools/web-format run check` で整形状態を確認できます。
