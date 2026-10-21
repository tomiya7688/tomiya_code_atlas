# GUI usage

Japanese source: [GUI operation guide](../jp/GUI操作ガイド.md).

Start the Windows `tomiya-code-atlas.exe` without arguments to open the desktop GUI. The Go GUI generates call graphs and class diagrams from one source file or a project folder. You can select both diagrams in one run. The `analyze` CLI command remains available.

## Generate diagrams

1. Choose **File** or **Project folder**, then click the matching selection button. `Alt+O` opens the current input picker.
2. Select one or both diagrams: **Call graph** and **Class diagram**.
3. Choose an output folder with **Choose output folder**. `Alt+P` opens its picker.
4. Click **Analyze and create diagrams**. `Alt+R` starts the analysis.
5. After completion, check the displayed Markdown output paths.

Errors appear in the GUI. Native dialogs select input and output paths. The Go application and its bundled parser helpers process source files locally; the application does not send selected source to an external service.

## Requirements and current scope

The GUI targets Windows 10/11 and uses the Microsoft Edge WebView2 Runtime. It is included in Windows 11 and many Windows 10 installations; install the WebView2 Runtime where it is missing. The Go GUI does not require Python or Tk at runtime. Language-specific parser helpers are included in the distribution.

The current Go GUI offers `call-graph` and `class-diagram`. Comment generation, editable preview, and other diagram types are tracked in later migration issues. Download the EXE from a successful [Go EXE workflow run](https://github.com/tomiya7688/tomiya_code_atlas/actions/workflows/go-exe.yml).

The CLI remains available:

```text
tomiya-code-atlas.exe analyze sample.py --operation call-graph --operation class-diagram --output-dir atlas-output
```

See the [CLI reference](CLI-Reference.md) for details.
