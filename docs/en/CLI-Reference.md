# Command-line reference

The Go distribution currently supports only help and version output while the port is in progress. The commands below describe the existing Python source implementation.

Japanese source: [README](../../README.md). This guide documents the commands currently exposed by `app.py`; run `python app.py <command> --help` for the live option list.

## Start here

From the repository root, use Python:

```text
python app.py --help
python app.py --version
```

The Go Windows executable currently supports only `--help` and `--version`. The following product commands are available only in the legacy Python source implementation; they are not yet available through `run_dist.bat`:

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
