# Build the Windows distribution

The Japanese source for this guide is the [Windows build section in the README](../../README.md) and the [development workflow guide](../jp/開発運用.md).

## Requirements

- Windows x64
- Python 3.11 or later (Python 3.12 is preferred)
- .NET SDK 10 x64
- JDK 25 x64
- An internet connection on the first build, so Python packages and Maven can be prepared

The build script detects the JDK through `JAVA_HOME` or `javac` on `PATH`; setting `JAVA_HOME` is optional when the JDK's `bin` folder is on `PATH`. If Apache Maven is not already available, the script downloads the pinned distribution into `.build\tools` and verifies its SHA-512 checksum.

## Build and run

Open a terminal in the repository root and run:

```bat
build_exe.bat
run_dist.bat
```

`build_exe.bat` prepares `.venv` when it is missing, builds the Java and C# analysis backends, and creates a PyInstaller `onedir` distribution. The output is:

```text
.build\dist\tomiya-code-atlas\
```

To run the packaged command-line application, pass arguments through the launcher:

```bat
run_dist.bat --help
run_dist.bat --version
```

To run the complete local build and verification sequence, including Python tests and packaged-artifact checks, use:

```bat
scripts\build\verify.bat
```

For source execution after setting up the environment, use `scripts\build\run_source.bat`. Setup and package helper scripts are grouped under `scripts\build\`; their working directory is resolved from the repository location.

If a prerequisite is missing, the build script reports the required SDK or installation link. See the [Japanese source guide](../../README.md) for the canonical instructions.
