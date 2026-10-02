# Tomiya Code Atlas

Tomiya Code Atlas analyzes source-code structure, behavior, dependencies, and responsibilities, then presents the results as diagrams, tables, comments, and design evaluations.

The Japanese documents are the source of truth. Start with the [Japanese README](../../README.md), or use these English guides:

- [Build the Windows distribution](Building.md)
- [Use the desktop GUI](GUI-Usage.md)
- [Use the command-line interface](CLI-Reference.md)

The tool currently targets Python, GDScript, C#, C++, Java, and Go. Feature support varies by command and language. See the [Japanese current-status document](../jp/現状.md) for the authoritative capability and limitation list.

The product is being ported to Go. The current Go Windows executable is only a buildable bootstrap; analysis features and the GUI have not been ported yet. The existing feature descriptions refer to the Python source implementation during migration. Track progress in [Issue #27](https://github.com/tomiya7688/tomiya_code_atlas/issues/27) and its child issues.

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
