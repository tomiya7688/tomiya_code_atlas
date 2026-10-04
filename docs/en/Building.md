# Build the Windows distribution

The Japanese source for this guide is the [Windows build section in the README](../../README.md) and the [development workflow guide](../jp/開発運用.md).

## Requirements

- Windows x64
- Go 1.22 or later to build the Go Windows executable
- .NET 10 SDK to publish the self-contained Roslyn C# helper
- Python 3.11 or later for Python source tests, reference runs, and the CPython parser helper build
- PyInstaller (installed by the repository setup script)

## Build and run

Open a terminal in the repository root and run:

```bat
build_exe.bat
run_dist.bat
```

`build_exe.bat` builds the Go CLI, a PyInstaller one-dir helper containing CPython, and a self-contained .NET 10 Roslyn helper. The current Go bootstrap supports help/version while product features are being migrated. The outputs are:

```text
.build\dist\tomiya-code-atlas.exe
.build\dist\backends\tomiya-python-backend\
.build\dist\backends\csharp\
```

To run the packaged command-line application, pass arguments through the launcher:

```bat
run_dist.bat --help
run_dist.bat --version
```

The Python source remains temporarily for reference and tests. The Python parser helper uses CPython's AST and ships in a one-dir folder with its runtime included. Its supported grammar follows the bundled interpreter version. To run the local verification sequence, including source tests and frozen helper smoke checks, use:

```bat
scripts\build\verify.bat
```

For Python source execution after setting up the environment, use `scripts\build\run_source.bat`. Setup and verification helpers are grouped under `scripts\build\`; their working directory is resolved from the repository location. All language backends use a real language parser or compiler AST rather than a hand-written scanner. The C# helper accepts Roslyn's preview grammar and includes its .NET runtime, so end users do not need to install .NET.

If Go is missing, the build script reports the required version and installation link. See the [Japanese source guide](../../README.md) for the canonical instructions.
