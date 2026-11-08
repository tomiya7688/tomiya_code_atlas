# GUI usage

Japanese source: [GUI operation guide](../jp/GUI%E6%93%8D%E4%BD%9C%E3%82%AC%E3%82%A4%E3%83%89.md).

Start the Windows `tomiya-code-atlas.exe` without arguments to open the desktop GUI. The Go GUI can analyze a file or project folder and generate diagrams, reports, or a commented source copy. You can select multiple outputs in one run. The `analyze` CLI command remains available.

## Generate analysis outputs

1. Choose **File** or **Project folder**, then click the matching selection button. `Alt+O` opens the current input picker.
1. Select one or more enabled outputs. Available outputs depend on the input type, source language, and whether the project contains workflow or deployment files.
1. Choose an output folder with **Choose output folder**. `Alt+P` opens its picker.
1. Click **Analyze and create results**. `Alt+R` starts the analysis.
1. After completion, check the displayed output file paths.

Errors appear in the GUI. Native dialogs select input and output paths. The Go application and its bundled parser helpers process source files locally; the application does not send selected source to an external service.

## Requirements and current scope

The GUI targets Windows 10/11 and uses the Microsoft Edge WebView2 Runtime. It is included in Windows 11 and many Windows 10 installations; install the WebView2 Runtime where it is missing. The Go GUI does not require Python or Tk at runtime. Language-specific parser helpers are included in the distribution.

The Go CLI registers 16 operation IDs, but not every input supports every operation. A single YAML file enables the CI graph. A source file enables operations supported by its parser and input type. Object, state, timing, and use-case diagrams currently require Python source. Package and component diagrams require a project folder containing source; deployment diagrams require deployment configuration; CI graphs require workflow files. Diagrams and reports are saved as Markdown files directly in the selected output folder. Commented source copies are saved under its `comments/` subfolder, preserving the input folder structure, without changing the original source. For `comments` on a project folder, choose an output folder whose `comments/` subfolder is not the same as, an ancestor of, or a descendant of the input folder. The app rejects an overlapping location before analysis. Editing the comment preview remains unsupported. Deployment output combines facts parsed from Dockerfile/Compose with inferred source-import relationships. Download the EXE from a successful [Go EXE workflow run](https://github.com/tomiya7688/tomiya_code_atlas/actions/workflows/go-exe.yml).

The CLI remains available:

```text
tomiya-code-atlas.exe analyze sample.py --operation call-graph --operation class-diagram --output-dir atlas-output
```

See the [CLI reference](CLI-Reference.md) for details.

## Python reference implementation

Developers can prepare the Python environment and run `python -m Src.app gui` to use the reference implementation. It is separate from the Go distribution.
