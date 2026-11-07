# Build the Windows distribution

The Japanese source for this guide is the [Windows build section in the README](../../README.md) and the [development workflow guide](../jp/%E9%96%8B%E7%99%BA%E9%81%8B%E7%94%A8.md).

## Requirements

- Windows x64
- Go 1.27.x to build the Go Windows executable and parser helper
- .NET 10 SDK to publish the self-contained Roslyn C# helper
- JDK 25 and Apache Maven to build the JavaParser helper and its private runtime
- Use stable CPython 3.11.18 or later in the 3.11 series, 3.12.16 or later in the 3.12 series, 3.13.18 or later in the 3.13 series, 3.14.10 or later in the 3.14 series, or 3.15.2 or later in the 3.15 series for Python source tests, reference runs, and the CPython parser helper build
- The setup security checker rejects CPython minor series without a configured fixed patch floor. Add a series only after verifying its official security release.
- On Windows, install CPython 3.14.10 or 3.15.2 or later with the Python install manager. The 3.11.18, 3.12.16, and 3.13.18 security releases are source-only and do not include Windows installers ([official release announcement](https://blog.python.org/2026/10/python-3152-31410-31318-31216-31118/)).
- PyInstaller (installed by the repository setup script)

## Build and run

Open a terminal in the repository root and run:

```bat
build_exe.bat
run_dist.bat
```

`build_exe.bat` builds the Go CLI and Go standard-parser helper, CPython/GDScript and C++ PyInstaller one-dir helpers, the self-contained .NET 10 Roslyn helper, and the JavaParser helper with a private jlink runtime containing `java.se` and required JDK modules. JDK 25 and Maven are needed only to build the Java helper; Python setup does not require them. The Go CLI scans a source file or project folder and generates call graphs and class diagrams. Folder files are parsed independently and their Common IR facts are combined; cross-file name and type resolution is not available yet. The outputs include:

```text
.build\dist\tomiya-code-atlas.exe
.build\dist\backends\tomiya-python-backend\
.build\dist\backends\csharp\
.build\dist\backends\cpp\
.build\dist\backends\java\
.build\dist\backends\go\tomiya-go-backend.exe
```

The Go EXE workflow uploads the `tomiya-code-atlas-go-windows-x64` artifact only after the CLI, GUI startup, and one-dir E2E smoke checks pass. Pull request events run only the build job, which has no `id-token` permission and does not publish an artifact. Pushes to the canonical repository (including pull request branches) and manual runs create an intermediate artifact, then a separate job creates and verifies GitHub OIDC provenance attestations for the ZIP, manifests, and bundled binaries before publishing the final artifact. Forks do not publish the final artifact. If a smoke check or attestation fails, the workflow does not upload the final distribution artifact.

Extract the GitHub Actions artifact and find the distribution ZIP and `SHA256SUMS.txt`. In an environment with GitHub CLI installed, use the source ref and commit SHA shown in the Actions run to verify ZIP provenance: `gh attestation verify tomiya-code-atlas-windows-x64.zip -R tomiya7688/tomiya_code_atlas --signer-workflow tomiya7688/tomiya_code_atlas/.github/workflows/go-exe.yml --source-ref <source-ref> --source-digest <commit-sha>`. A branch push uses a ref such as `refs/heads/<branch>`; a tag push uses `refs/tags/<tag>`. Replace `<source-ref>` and `<commit-sha>` with the run's values. The `SHA256SUMS.txt` inside the ZIP lets you verify the extracted parser binaries. This process verifies artifacts produced by this workflow; older artifacts do not have provenance from it. Provenance attestations establish the build source and integrity; they do not add a Windows Authenticode Publisher signature. Authenticode signing for official releases remains unavailable until a trusted signing certificate is provisioned.

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

Before creating a virtual environment or updating pip, the setup script checks the selected CPython interpreter against the stable patch floors with `tools/python_tls_floor.py`. It stops before package network access when the interpreter is unsupported. It then updates pip and certifi from SHA-256-pinned wheels and checks package versions and trust stores before installing other packages. It compares the installed and imported versions of pip and certifi and checks pip's vendored certifi version and bundle. It also inspects pip's configured CA overrides from `PIP_CERT`, `REQUESTS_CA_BUNDLE`, `CURL_CA_BUNDLE`, and `cert` entries in `pip.ini` / `pip.conf`; it searches the documented Windows, Linux, and macOS configuration locations. It follows pip's precedence rule: when `PIP_CONFIG_FILE` points to an existing file, user-level configuration is omitted while global and site configuration remain active. Use stable CPython releases at or above these patch levels for each minor series: 3.11.18, 3.12.16, 3.13.18, 3.14.10, and 3.15.2. Patched CPython rejects the compromised intermediate keys. The checker also requires the affected emSign root to be absent from the OS trust store, including certificates in OpenSSL's default and `SSL_CERT_DIR` hashed CA directories, and stops setup if it remains. On Windows, apply Windows Update and rerun `scripts\build\setup.bat` after this check fails. The Python parser helper currently makes no network requests. Its one-dir package includes a certifi CA bundle for future TLS use, and the build checks that bundle; this check does not mean the helper currently selects it for TLS connections. Run `scripts\build\setup.bat` to update an existing repository `.venv` as well as prepare a new one.

On Windows, when the Python Launcher is available, setup checks installed runtimes in descending minor-version order (3.15, 3.14, 3.13, 3.12, then 3.11) and selects the first stable CPython that meets its patch floor. If none passes, setup checks the `python` command on `PATH`.

The Java helper uses JavaParser's explicit Java 26 grammar level. Its distribution contains a private Java runtime built from JDK 25, so end users do not need Java, a JDK, or Maven.

This repository does not use `uv` or `sigstore`. If either becomes a dependency, use at least `uv` 0.13.6 and `sigstore` 4.5.1, as listed in the [Python security incident announcement](https://discuss.python.org/t/information-for-python-users-regarding-emudhra-emsign-ca-certificate-security-incident/109576).

If Go is missing, the build script reports the required version and installation link. See the [Japanese source guide](../../README.md) for the canonical instructions.
