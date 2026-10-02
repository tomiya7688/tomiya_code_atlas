# Build the Windows distribution

The Japanese source for this guide is the [Windows build section in the README](../../README.md) and the [development workflow guide](../jp/開発運用.md).

## Requirements

- Windows x64
- Go 1.22 or later to build the Go Windows executable
- Python 3.11 or later only for Python source tests and reference runs

## Build and run

Open a terminal in the repository root and run:

```bat
build_exe.bat
run_dist.bat
```

`build_exe.bat` builds the Go Windows executable. The current Go bootstrap supports help/version while product features are being migrated. The output is:

```text
.build\dist\tomiya-code-atlas.exe
```

To run the packaged command-line application, pass arguments through the launcher:

```bat
run_dist.bat --help
run_dist.bat --version
```

The Python source remains temporarily for reference and tests; no Python executable or package is built. To run the local verification sequence, including source tests and the Go executable smoke checks, use:

```bat
scripts\build\verify.bat
```

For Python source execution after setting up the environment, use `scripts\build\run_source.bat`. Setup and verification helpers are grouped under `scripts\build\`; their working directory is resolved from the repository location.

If Go is missing, the build script reports the required version and installation link. See the [Japanese source guide](../../README.md) for the canonical instructions.
