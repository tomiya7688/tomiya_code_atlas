# Diagram Series Partitioning

The authoritative specification is [the Japanese version](../jp/機能仕様/図の系列分割.md).

## Purpose

Tomiya Code Atlas splits large relationship diagrams into smaller views. A split must not hide semantic relationships. The application builds the complete semantic graph first, then partitions only its presentation. Partitioning never changes the original nodes, relationships, or UML meaning.

## Visible behavior

When analysis runs, the application keeps the complete graph in metadata and creates diagrams with boundary stubs. Each stub names the diagram that continues the relationship. High-fan-in nodes are placed in shared diagrams, and regular diagrams keep references to those shared views. Placement metadata retains each series root, parent root, and depth. Combining relationships across all outputs restores the original graph.

## Partition rules

1. Nodes at or above the configured fan-in threshold are placed in shared diagrams.
2. Remaining nodes are grouped into weakly connected components.
3. Each component is organized around caller-side roots. Closed child branches are recursively split only when a parent branches.
4. A branch below the configured minimum child-series size remains in its parent. Simple chains are kept together.
5. Nodes in the same strongly connected component (SCC) remain in one series. Cyclic regions are not split.
6. The partition metadata retains every original edge. A relationship crossing a series or shared boundary appears with an external stub and a destination diagram reference.
7. Relationships connected to shared nodes can be followed from both regular and shared diagrams.
8. Series hierarchy remains traceable through root, parent root, and depth metadata.

## Configuration

- Callers provide the fan-in threshold.
- The partition API accepts the minimum child-series size. Diagram generators use a default of three nodes.
- A maximum diagram node count is a future option and is not part of current behavior.

Changing thresholds affects only presentation partitioning and its metadata.

## Implementation and verification

`go/internal/analyzers` stores all input edges in `GraphPartition.OriginalEdges` and records boundary ownership in `BoundaryEdges`. Generators add stubs and destination names to renderer-neutral models; renderers display those references.

From `go/`, run `go test ./internal/analyzers ./internal/generators ./internal/renderers`. The tests cover cycles, fan-in, branch splitting, retained boundary edges, and rendered destination references. Review Mermaid and PlantUML output to confirm that stubs and continuation labels are visible. To inspect the packaged Windows executable, run `build_exe.bat` from the repository root, then run `\.venv\Scripts\python.exe tools\go_cli_smoke.py .build\dist\tomiya-code-atlas.exe`. This smoke test also verifies Go CLI partitioning and boundary relations; `.github/workflows/go-exe.yml` runs it before publishing the distribution artifact.
