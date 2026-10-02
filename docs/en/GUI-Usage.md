# GUI usage

Japanese source: [GUI operation guide](../jp/GUI操作ガイド.md).

The existing Python source implementation opens its desktop GUI when you start `app.py` without arguments. The Go Windows executable currently supports only help and version; it does not yet include the GUI or analysis features.

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
python app.py ci .github/workflows/ci.yml
python app.py --version
python app.py gui
```

See the [CLI reference](CLI-Reference.md) for command options. The Japanese guide is the source of truth for GUI capabilities and limitations.
# Migration status

This guide describes the existing Python source GUI. The Go Windows executable does not include analysis features or a GUI yet.
