# GUIの使い方

英語版: [GUI Usage](../en/GUI-Usage.md)

既存Python source版は、`app.py` を引数なしで起動するとデスクトップGUIを開きます。

初期GUIで利用できる機能:

- 対応ファイルまたはプロジェクトフォルダーの選択
- 対応ファイル一覧と検出言語の表示
- コメント生成
- PythonコールグラフのMermaid生成
- Pythonクラス責務表のMarkdown生成
- GitHub Actions YAMLのCIグラフ生成
- 結果ソースの確認と保存

CLIも引き続き利用できます。

```text
python app.py comment sample.py
python app.py ci .github/workflows/ci.yml
python app.py --version
python app.py gui
```

現段階のMermaidプレビューはMermaidソースの表示です。画像やHTMLとしての埋め込みレンダリングは今後の拡張です。

GUI層は `Src/ui/` に限定し、解析処理は `Src/process/application.py` を経由して呼び出します。
# 移行中の注意

このガイドは既存Python source版のGUIを対象にしています。Go版配布EXEには、まだ解析機能とGUIが移植されていません。Go版EXEで現在使えるのはhelp/versionのみです。
