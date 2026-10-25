# Go GUI usage

Japanese source: [GUI operation guide](../jp/GUI操作ガイド.md).

The Go GUI lets Windows users choose source files or project folders, select diagrams, set an output folder, and review status, diagnostics, and generated files in one window. The GUI runs in Go and does not require Python to be installed. Language parsers start from the bundled `backends/` directory.

Double-click `tomiya-code-atlas.exe` in the Windows distribution to start the GUI. Developers can run `run_dist.bat` without arguments from the repository root.

## Run an analysis

1. Choose **Select file** or **Select folder** and select a source file or project folder. You can also type a path directly. Relative paths use the working directory from which the app was started. Use the picker buttons if you are unsure of that location.
2. Check either or both diagrams under **Choose diagrams**. The GUI lists only diagrams currently available in the Go version.
3. Confirm the output folder. The default is `Tomiya Code Atlas/atlas-output` under your home folder. Choose another folder or type a path to change it.
4. Select **Run analysis**. The GUI shows the running status. When analysis finishes, it lists the generated file paths. On failure, it shows the cause under **Diagnostics**.

Keyboard shortcuts are also available: `Alt+F` selects a source file, `Alt+P` selects an input folder, `Alt+O` selects an output folder, and `Alt+R` runs the selected analyses. Each button displays its shortcut.

## Current support

The Go GUI currently generates Markdown call graphs and class diagrams. For a project folder, it scans supported source files recursively and sends each file to its parser. Cross-file name and type resolution, comment generation, other diagrams, and CI analysis are still being ported. Unsupported diagrams are not shown in the GUI.

For command-line use, run `tomiya-code-atlas-cli.exe` or pass CLI arguments to `run_dist.bat`. See the [CLI reference](CLI-Reference.md).

Developers can run `build_exe.bat` from the repository root to build the GUI, CLI, and parser helpers as a directory-based distribution.

Visual interaction and execution from the GitHub distribution artifact are still being accepted.
