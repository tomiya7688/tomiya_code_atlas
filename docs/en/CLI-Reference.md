# Command-line reference

The Go Windows distribution includes a desktop GUI and a separate console CLI. Both support recursive project scanning and call-graph and class-diagram generation. Each discovered source file is parsed independently and its Common IR facts are combined; cross-file name and type resolution is not available yet. Other operations remain in progress. See the [GUI guide](GUI-Usage.md) and the [Japanese Go CLI guide](../jp/CLI利用ガイド.md) for canonical instructions.

Japanese source: [README](../../README.md). This guide documents the commands currently exposed by `Src.app`; run `python -m Src.app <command> --help` for the live option list.

## Start here

From the repository root, use Python:

```text
python -m Src.app --help
python -m Src.app --version
```

The Go CLI command is available through `run_dist.bat` or `tomiya-code-atlas-cli.exe`:

```bat
run_dist.bat analyze sample.py --operation call-graph --operation class-diagram --output-dir atlas-output
```

The current Go command accepts one source file or project folder and requires at least one `--operation` and an `--output-dir`. A folder is scanned recursively. Each selected operation creates one Markdown file in that folder. The parser helper and `backends/manifest.json` must be next to the executable. Run `run_dist.bat analyze --help` for the option list.

When a parser fails, the CLI shows a Japanese cause and a suggested next step. The `Details:` portion preserves the parser's technical message.

The following broader commands remain available only in the legacy Python source implementation:

```bat
python -m Src.app --help
python -m Src.app comment sample.py
```

The Go desktop GUI opens when `tomiya-code-atlas.exe` is launched without arguments. `run_dist.bat` also opens it without arguments; CLI arguments launch the console sidecar. The legacy Python source application remains available through `python -m Src.app gui`.

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
