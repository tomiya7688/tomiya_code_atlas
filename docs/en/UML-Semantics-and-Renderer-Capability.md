# UML Semantics and Renderer Capabilities

## Purpose

Tomiya Code Atlas converts analysis results into logical diagram models, then serializes those models as Mermaid or PlantUML. A generator model is independent of source languages and output syntax. It is a product-level representation for the documented use cases, not a complete UML 2.x metamodel.

A renderer maps a logical model to a target notation. When the target has no native diagram syntax for that model, the renderer may use an approximation only when the Capability Matrix identifies it as such. An approximation must not be presented as UML-conformant output.

## Semantic boundary

```text
Language parser
  -> language-neutral analysis results
  -> logical diagram model
  -> presentation partition into series/shared views
  -> renderer-specific syntax
```

- Analyzers and generators do not store Mermaid or PlantUML syntax, node IDs, or arrow notation in logical models.
- Models retain the elements, relationships, ordering, and conditions produced from source analysis. They also retain confirmed/inferred distinctions when available.
- Partitioning creates presentation views. It does not delete or alter semantic relationships; boundaries are navigational references. See [Diagram series partitioning](Series-Partition.md).
- Cycle highlighting, confidence, design warnings, high fan-in, shared groups, and continuation references are Tomiya Code Atlas overlays or navigation metadata. They do not change UML semantics.
- A renderer does not reinterpret one relationship as another. If a notation cannot represent model semantics, the capability is marked approximate or unsupported.

## Capability Matrix

The status describes the current Go implementation. “Faithful” means that the target diagram notation can express the current logical model fields; it does not claim full conformance to the UML specification. “Approximate” means the output uses another diagram grammar or omits UML semantics. “Unsupported” means the project has no renderer for that format and diagram type.

| Diagram type  | Go logical model       | Mermaid                        | PlantUML                   | Current semantics and limits                                                                                                                                                                                                                                                                  |
| ------------- | ---------------------- | ------------------------------ | -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Class         | `ClassDiagram`         | Approximate: `classDiagram`    | Approximate: class diagram | Classes/members, visibility, and inheritance. Calls produce `uses` relations that both renderers map to association `-->`, so this UML relation is approximate. Other kinds are omitted. See [#209](https://github.com/tomiya7688/tomiya_code_atlas/issues/209).                              |
| Sequence      | `SequenceDiagram`      | Faithful: `sequenceDiagram`    | Faithful: sequence diagram | Ordered messages, calls, returns, async and recursive calls, and depth truncation. UML interaction fragments, activations, and lifeline creation/destruction are not represented. See [#210](https://github.com/tomiya7688/tomiya_code_atlas/issues/210).                                     |
| State machine | `StateDiagram`         | Approximate: `stateDiagram-v2` | Unsupported                | Initial/terminal states, transitions, events, and guards are represented. A wildcard source is rendered as a synthetic “any current state”; composite states, regions, and entry/exit/do actions are not represented. See [#211](https://github.com/tomiya7688/tomiya_code_atlas/issues/211). |
| Activity      | `ActivityFlow`         | Approximate: `flowchart`       | Unsupported                | Action-like nodes, decisions, and ordered edges. UML activity partitions, fork/join, and object flows are not guaranteed. See [#219](https://github.com/tomiya7688/tomiya_code_atlas/issues/219).                                                                                             |
| Communication | `CommunicationDiagram` | Approximate: `flowchart`       | Unsupported                | Participant relationships and message order labels. UML connectors/message semantics and ordering equivalent to a sequence diagram are not guaranteed. See [#212](https://github.com/tomiya7688/tomiya_code_atlas/issues/212).                                                                |
| Object        | `ObjectDiagram`        | Approximate: `flowchart`       | Unsupported                | Instances, types, field values, and references as node/edge labels. Formal UML `InstanceSpecification`, slots, and classifier semantics are not guaranteed. See [#213](https://github.com/tomiya7688/tomiya_code_atlas/issues/213).                                                           |
| Package       | `PackageDiagram`       | Approximate: `flowchart`       | Unsupported                | Packages/modules, dependency edges, and cycle information. UML package import and merge are not represented. See [#214](https://github.com/tomiya7688/tomiya_code_atlas/issues/214).                                                                                                          |
| Use case      | `UseCaseDiagram`       | Approximate: `flowchart`       | Unsupported                | Actors and use cases are drawn; handler/call chain appear only in Mermaid source comment `%% handler-chain`, not images. `usecase-beta` and include/extend are unsupported. See [#215](https://github.com/tomiya7688/tomiya_code_atlas/issues/215).                                           |
| Component     | `ComponentDiagram`     | Approximate: `flowchart`       | Unsupported                | Components, dependencies, external/boundary nodes, and cycles. UML ports, provided/required interfaces, and assembly connectors are not represented. See [#216](https://github.com/tomiya7688/tomiya_code_atlas/issues/216).                                                                  |
| Deployment    | `DeploymentDiagram`    | Approximate: `flowchart`       | Unsupported                | Workloads, images, databases, node kinds, connections, and confidence. Full UML node, deployment specification, and communication path semantics are not guaranteed. See [#217](https://github.com/tomiya7688/tomiya_code_atlas/issues/217).                                                  |
| Timing        | `TimingChart`          | Approximate: `sequenceDiagram` | Unsupported                | Ordered processing events, await, parallel, and wait. There is no time axis or time-varying lifeline state/value. See [#218](https://github.com/tomiya7688/tomiya_code_atlas/issues/218).                                                                                                     |
| Call graph    | `CallGraph`            | Product graph: `flowchart`     | Unsupported                | Caller/callee relationships. The product does not claim this is a UML diagram.                                                                                                                                                                                                                |

Even “faithful” entries are limited to the model fields described above. The renderer must not imply that unrepresented UML concepts exist by omitting them or substituting another relation or label.

## Output and partitioning

Artifact names identify diagram types; they do not claim full UML conformance. The matrix documents each approximation and its omitted meaning. A renderer must not silently substitute another syntax for an unsupported diagram type.

Shared nodes, boundary stubs, and continuation references help users navigate partitioned views. They are presentation metadata. The original semantic model and partition metadata retain relationships even when a diagram boundary cannot display the full relationship directly.

## Golden tests

The Go [shared renderer golden test](../../go/internal/renderers/renderers_test.go) compares representative class, sequence, communication, object, package, component, state, timing, use-case, deployment, and call-graph output with a fixture shared with Python. Activity flowchart output has a dedicated renderer test. These goldens verify stable syntax; they do not prove full UML conformance or the readability of a rendered image.

When a related issue expands model semantics, update the capability matrix, renderer expectations, and user-facing documentation together. Review rendered images as well as source goldens when visual acceptance is required.

## Sources and maintenance

The Go renderer implementation and golden tests define the current project capability. Mermaid and PlantUML syntax can change by version, so issues that change output syntax must verify the official syntax references and the renderer version used:

- [Mermaid diagram syntax](https://mermaid.js.org/intro/syntax-reference.html)
- [PlantUML diagram overview](https://plantuml.com/)
- [PlantUML class diagrams](https://plantuml.com/class-diagram)
- [PlantUML sequence diagrams](https://plantuml.com/sequence-diagram)

## How to verify

After changing a diagram renderer or logical model, developers should run `go test ./...` from `go/` and review shared golden diffs. When UML conformance or visual readability is part of acceptance, golden tests alone are insufficient; inspect the rendered output.
