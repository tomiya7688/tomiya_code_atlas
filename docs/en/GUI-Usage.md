# GUI usage

Japanese source: [GUI operation guide](../jp/GUI操作ガイド.md).

Start `app.py` or the packaged `tomiya-code-atlas.exe` without arguments to open the desktop GUI. In the distribution folder, you can double-click the executable or run `run_dist.bat` from the repository root.

The current GUI can:

- Select supported files or a project folder.
- Show discovered files and detected languages.
- Generate source comments.
- Generate a Mermaid call graph for Python.
- Generate a Markdown class-responsibility table for Python.
- Generate a CI graph from a GitHub Actions YAML file.
- Preview the generated source and save the result.

The Mermaid preview currently displays Mermaid source text. It does not render the diagram as an image or embedded HTML.

CLI commands remain available alongside the GUI. For example:

```text
python app.py comment sample.py
python app.py ci .github/workflows/build.yml
python app.py --version
python app.py gui
```

See the [CLI reference](CLI-Reference.md) for command options. The Japanese guide is the source of truth for GUI capabilities and limitations.
