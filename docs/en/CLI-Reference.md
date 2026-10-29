# Command-line reference

The Go CLI supports recursive project scanning, call graphs, class diagrams, commented source copies, static specifications, and design-quality reports. Each file is parsed independently; cross-file name and type resolution is not available yet. The Windows executable opens a GUI for call graphs and class diagrams without arguments. See the [Japanese Go CLI guide](../jp/CLI利用ガイド.md) and the [GUI guide](GUI-Usage.md).

The [Japanese Go CLI guide](../jp/CLI利用ガイド.md) is the canonical guide for the Go CLI. The Python reference implementation has separate commands described below.

## Start here

The Go CLI command is available through `run_dist.bat` or the packaged executable:

```bat
run_dist.bat analyze sample.py --operation call-graph --operation class-diagram --operation comments --operation static-spec --operation design-quality --output-dir atlas-output
```

The current Go command accepts one source file or project folder and requires at least one `--operation` and an `--output-dir`. A folder is scanned recursively. Diagram, specification, and quality outputs are Markdown files in the output folder. The `comments` operation writes language-specific comments into copies under `comments/`; directory inputs keep their relative paths, and a single file is named `<file>_commented.<extension>`. The input files are not modified. For directory input with `comments`, `<output-dir>/comments` and the input folder must be separate, non-overlapping trees: neither path can equal, contain, or be nested under the other. The CLI rejects overlapping paths before scanning to prevent generated copies from being rediscovered and to keep inputs separate from outputs. Use a separate output folder such as `atlas-output`. The parser helper and `backends/manifest.json` must be next to the executable. Run `run_dist.bat analyze --help` for the option list.

When a parser fails, the CLI shows a Japanese cause and a suggested next step. The `Details:` portion preserves the parser's technical message.

The following legacy command forms remain available only in the Python source implementation. The Go equivalents for comment generation and static specifications use `analyze --operation comments` and `analyze --operation static-spec`:

```bat
python -m Src.app --help
python -m Src.app comment sample.py
```

With no command, the application opens the desktop GUI. `python -m Src.app gui` opens it explicitly.

## Commands

### Generate comments

```text
python -m Src.app comment sample.py
python -m Src.app comment sample.cs --language csharp --output commented.cs
python -m Src.app comment sample.py --in-place
```

The language adapter is inferred from the file extension unless `--language` is supplied. `--output` writes to a separate file; `--in-place` updates the source file.

### Analyze a GitHub Actions workflow

```text
python -m Src.app ci .github/workflows/ci.yml
python -m Src.app ci .github/workflows/ci.yml --output workflow.md
python -m Src.app ci .github/workflows/ci.yml --check
```

`--check` reports quality findings and exits with failure when errors are found.

### Generate diagrams

```text
python -m Src.app call-graph Src --output-dir out
python -m Src.app class-diagram Src --renderer mermaid --output-dir out
python -m Src.app sequence-diagram Src --show-returns --output-dir out
python -m Src.app timing Src --output-dir out
python -m Src.app use-cases Src --output-dir out
python -m Src.app deployment . --mode full --output-dir out
```

Available diagram commands are `call-graph`, `class-diagram`, `sequence-diagram`, `timing`, `use-cases`, and `deployment`. Source-based commands accept `--language` where applicable. `--output-dir` selects an output folder; `class-diagram` and `sequence-diagram` also accept `--renderer mermaid` or `--renderer plantuml`.

Command-specific options include:

- `call-graph`: `--fan-in-threshold`, `--root`, and `--max-depth`.
- `sequence-diagram`: `--max-depth`, `--hide-duplicate-calls`, and `--show-returns`.
- `use-cases`: `--max-depth`.
- `deployment`: `--mode simple` or `--mode full`.

Run a command's help for its exact syntax and defaults, for example `python -m Src.app call-graph --help`.

### Generate a static specification

```text
python -m Src.app static-spec sample.py
python -m Src.app static-spec sample.py --output sample-spec.md
```

This documents one Python file or recursively scans a Python project (excluding generated and development-only folders), including declarations, available annotations, docstrings, source lines, and same-file statically resolved calls.

## Japanese source

The Japanese README is authoritative: [../../README.md](../../README.md). English guides are translations and must not define different behavior.
