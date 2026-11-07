# Tomiya Code Atlas

Tomiya Code Atlas analyzes source-code structure, behavior, dependencies, and responsibilities, then presents the results as diagrams, tables, comments, and design evaluations.

The Japanese documents are the source of truth. Start with the [Japanese README](../../README.md), or use these English guides:

- [Build the Windows distribution](Building.md)
- [Use the desktop GUI](GUI-Usage.md)
- [Use the command-line interface](CLI-Reference.md)
- [Diagram series partitioning](Series-Partition.md)
- [UML semantics and renderer capabilities](UML-Semantics-and-Renderer-Capability.md)

The tool currently targets Python, GDScript, C#, C++, Java, and Go. Feature support varies by command and language. See the [Japanese current-status document](../jp/%E7%8F%BE%E7%8A%B6.md) for the authoritative capability and limitation list.

The product is being ported to Go. The Windows executable opens a Japanese GUI for call graphs and class diagrams when started without arguments. The Go CLI recursively scans project folders and generates call graphs, class diagrams, sequence diagrams, commented source copies, static specifications, and design-quality reports from per-file parser results. Comments are written to copies without changing inputs. Cross-file name and type resolution and other GUI operations remain in progress. See the [CLI reference](CLI-Reference.md), [GUI guide](GUI-Usage.md), and canonical [Japanese CLI guide](../jp/CLI利用ガイド.md). Migration progress is tracked in [Issue #27](https://github.com/tomiya7688/tomiya_code_atlas/issues/27) and its child issues.

## Download and verify the Windows artifact

Open a successful `Go EXE` workflow run in the canonical GitHub repository and download its `tomiya-code-atlas-go-windows-x64` artifact. Pull request events and forks do not publish the final distribution artifact. Extract the ZIP, then use the branch name and commit SHA shown by the Actions run to verify its provenance with GitHub CLI:

```powershell
gh attestation verify tomiya-code-atlas-windows-x64.zip -R tomiya7688/tomiya_code_atlas --signer-workflow tomiya7688/tomiya_code_atlas/.github/workflows/go-exe.yml --source-ref <source-ref> --source-digest <commit-sha>
```

Replace `<source-ref>` and `<commit-sha>` with the values shown by the workflow run. A branch push uses a ref such as `refs/heads/<branch>`; a tag push uses `refs/tags/<tag>`. Use the `SHA256SUMS.txt` inside the ZIP to verify the extracted parser binaries. This verifies artifacts produced by this workflow; older artifacts do not have its provenance. Provenance does not add a Windows Authenticode signature. Official Authenticode signing remains unavailable until a trusted signing certificate is provisioned.

## Architecture

```text
Source
  -> Language Adapter / Parser
  -> Common IR / shared models
  -> Analyzer / Generator / Evaluator
  -> Logical Output
  -> Renderer
  -> Mermaid / PlantUML / text / table
```

Language-specific parser details stay behind adapters. Shared analysis, generation, and rendering remain replaceable. The normative architecture rules are maintained in [`specification/architecture-policy.md`](../../specification/architecture-policy.md).

## License

The source code is licensed under the [MIT License](../../LICENSE). The Atlas-kun images and derived application icons are separately licensed under the [Tomiya Character License v1.0.1](../../assets/characters/atlas-kun/LICENSE.md). See the [asset notice](../../assets/characters/atlas-kun/README.md) for source and distribution details.
