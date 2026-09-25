# Python GUI usage

Tomiya Code Atlas の Python + PyInstaller 実装では、`app.py` または生成した `tomiya-code-atlas.exe` を引数なしで起動するとデスクトップGUIを開く。

初期GUIで利用できる機能:

- 対応ファイルまたはプロジェクトフォルダの選択
- 対応ファイル一覧と検出言語の表示
- コメント生成
- PythonコールグラフのMermaid生成
- Pythonクラス責務表のMarkdown生成
- GitHub Actions YAMLのCIグラフ生成
- 結果ソースの確認と保存

CLIは引き続き利用可能。

```text
python app.py comment sample.py
python app.py ci .github/workflows/build.yml
python app.py --version
python app.py gui
```

現段階のMermaidプレビューはMermaidソース表示。画像/HTMLとしての埋め込みレンダリングは後続拡張とする。

GUI層は `Src/ui/` に限定し、解析処理は `Src/process/application.py` 経由で呼び出す。
