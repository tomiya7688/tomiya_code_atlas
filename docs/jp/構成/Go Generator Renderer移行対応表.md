# Go Generator / Renderer 移行対応表

Issue #196 で、共通IRまたは共通Analyzerの結果から論理出力を作る Generator と、その論理出力を文字列化する Renderer を Go へ移す。Renderer は parser / language adapter を参照せず、Generator が作った受動的modelだけを入力にする。

## 1.0 parity

| 1.0 出力 | Go Generator | Go Renderer | 形式 |
|---|---|---|---|
| Call graph | `go/internal/generators` | `RenderCallGraphMermaid` | Mermaid |
| Class diagram | `go/internal/generators` | `RenderClassDiagramMermaid` / `RenderClassDiagramPlantUML` | Mermaid / PlantUML |
| Object diagram | `go/internal/generators` | `RenderObjectDiagramMermaid` | Mermaid |
| Package diagram | `go/internal/generators` | `RenderPackageDiagramMermaid` | Mermaid |
| Component diagram | `go/internal/generators` | `RenderComponentDiagramMermaid` | Mermaid |
| Sequence diagram | `go/internal/generators` | `RenderSequenceDiagramMermaid` / `RenderSequenceDiagramPlantUML` | Mermaid / PlantUML |
| Communication diagram | `go/internal/generators` | `RenderCommunicationDiagramMermaid` | Mermaid |
| State diagram | `go/internal/generators` | `RenderStateDiagramMermaid` | Mermaid |
| Timing chart | `go/internal/generators` | `RenderTimingChartMermaid` | Mermaid |
| Use case diagram | `go/internal/generators` | `RenderUseCaseDiagramMermaid` | Mermaid |
| Deployment diagram | `go/internal/generators` | `RenderDeploymentDiagramMermaid` | Mermaid |
| Responsibility table | `go/internal/generators` | `RenderResponsibilityMarkdown` / `RenderResponsibilityCSV` | Markdown / CSV |
| Activity diagram | `go/internal/generators` | `RenderActivityFlowchart` | Mermaid |

## 境界

- Generator は Common IR / Analyzer の公開結果だけを読み、Mermaid・PlantUML・CSV・Markdown構文を生成しない。
- Renderer は `go/internal/generators` の論理modelだけを読み、Common IR、Analyzer、parser、language adapterへ依存しない。
- partitioned output は共通 `PartitionGraph` の series/shared と `OutputPlacement` を使い、series root / depth / parentを保持する。
- stable output name、node順、message順、末尾改行は決定的にする。

## 共有golden

Activity flowchart は `tests/fixtures/generators/activity_flowchart_v1.json` を Python / Go で共有し、node形状、edge順、方向、末尾改行を検証する。

`tests/fixtures/generators/renderers_v1.json` はPython 1.0とGo rendererが共有するgoldenで、Mermaid / PlantUML / Markdown / CSVの順序、escaping、改行を固定する。Go側は `go/internal/renderers/renderers_test.go`、Python側は `tests/test_generator_renderer_shared_fixture.py` で同じ期待値を検証する。

State / Timing logical modelは既存の `tests/fixtures/analyzers/state_timing_v1.json` をPython/Goで共有している。その他Generatorは #195 で移植済みの関係graph、call sequence、package/component/deployment topologyを再利用し、同じpartition規則から論理modelを構築する。

## このIssueで扱わないもの

- static specification、comment specification、evaluator は #197。
- CLI / Process / Data / config / project scan の配線は #198。
- GUIは #199。
- release / packaging parityは #200。
