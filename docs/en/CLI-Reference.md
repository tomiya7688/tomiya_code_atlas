# Command-line reference

The Go distribution supports recursive project scanning and call-graph and class-diagram generation. Each discovered source file is parsed independently and its Common IR facts are combined; cross-file name and type resolution is not available yet. Other operations and the GUI are still being ported. See the [Japanese Go CLI guide](../jp/CLI利用ガイド.md) for the canonical instructions.

Japanese source: [README](../../README.md). This guide documents the commands currently exposed by `app.py`; run `python app.py <command> --help` for the live option list.

## Start here

From the repository root, use Python:

```text
python app.py --help
python app.py --version
```

The Go CLI command is available through `run_dist.bat` or the packaged executable:

```bat
run_dist.bat analyze sample.py --operation call-graph --operation class-diagram --output-dir atlas-output
```

The current Go command accepts one source file or project folder and requires at least one `--operation` and an `--output-dir`. A folder is scanned recursively. Each selected operation creates one Markdown file in that folder. The parser helper and `backends/manifest.json` must be next to the executable. Run `run_dist.bat analyze --help` for the option list.

The following broader commands remain available only in the legacy Python source implementation:

```bat
python app.py --help
python app.py comment sample.py
```

With no command, the application opens the desktop GUI. `python app.py gui` opens it explicitly.

## Commands

### Generate comments

```text
python app.py comment sample.py
python app.py comment sample.cs --language csharp --output commented.cs
python app.py comment sample.py --in-place
```

The language adapter is inferred from the file extension unless `--language` is supplied. `--output` writes to a separate file; `--in-place` updates the source file.

### Analyze a GitHub Actions workflow

```text
python app.py ci .github/workflows/ci.yml
python app.py ci .github/workflows/ci.yml --output workflow.md
python app.py ci .github/workflows/ci.yml --check
```

`--check` reports quality findings and exits with failure when errors are found.

### Generate diagrams

```text
python app.py call-graph Src --output-dir out
python app.py class-diagram Src --renderer mermaid --output-dir out
python app.py sequence-diagram Src --show-returns --output-dir out
python app.py timing Src --output-dir out
python app.py use-cases Src --output-dir out
python app.py deployment . --mode full --output-dir out
```

Available diagram commands are `call-graph`, `class-diagram`, `sequence-diagram`, `timing`, `use-cases`, and `deployment`. Source-based commands accept `--language` where applicable. `--output-dir` selects an output folder; `class-diagram` and `sequence-diagram` also accept `--renderer mermaid` or `--renderer plantuml`.

Command-specific options include:

- `call-graph`: `--fan-in-threshold`, `--root`, and `--max-depth`.
- `sequence-diagram`: `--max-depth`, `--hide-duplicate-calls`, and `--show-returns`.
- `use-cases`: `--max-depth`.
- `deployment`: `--mode simple` or `--mode full`.

Run a command's help for its exact syntax and defaults, for example `python app.py call-graph --help`.

### Generate a static specification

```text
python app.py static-spec sample.py
python app.py static-spec sample.py --output sample-spec.md
```

This documents one Python file or recursively scans a Python project (excluding generated and development-only folders), including declarations, available annotations, docstrings, source lines, and same-file statically resolved calls.

## Japanese source

The Japanese README is authoritative: [../../README.md](../../README.md). English guides are translations and must not define different behavior.
