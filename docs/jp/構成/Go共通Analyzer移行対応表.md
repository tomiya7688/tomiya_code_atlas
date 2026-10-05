# Go共通Analyzer移行対応表

Issue #195で、Python版の決定的な共通解析をCommon IR入力のGo実装へ段階的に移します。Go Analyzerは言語parserやRendererに依存せず、論理modelだけを返します。

## 対応状況

| Analyzer | Python版 | Go版 | 比較根拠 |
| --- | --- | --- | --- |
| Call graphの構築、fan-in/out、到達範囲、深さ制限、循環 | `Src/analyzers/call_graph.py` | `go/internal/analyzers/call_graph.go` | `tests/fixtures/backend_conformance/serialized_common_ir_v1.json` 内の`logical_output.call_graph`とGoテスト |
| package/module依存、相対import、孤立node、循環 | `Src/analyzers/package_dependencies.py` | `go/internal/analyzers/package_dependencies.go` | `tests/fixtures/analyzers/package_dependencies_v1.json`をPython/Go両方で読む |
| class relations（継承、解決済みclass use、外部base保持） | `Src/analyzers/class_relations.py` | `go/internal/analyzers/relations.go` | `tests/fixtures/analyzers/relations_v1.json` |
| object references（scope内解決、曖昧参照の除外、cycle） | `Src/analyzers/object_relations.py` | `go/internal/analyzers/relations.go` | `tests/fixtures/analyzers/relations_v1.json` |
| component依存（project内module集約、external package、cycle） | `Src/analyzers/component_dependencies.py` | `go/internal/analyzers/relations.go` | `tests/fixtures/analyzers/relations_v1.json` |
| partition、graph metrics、flow | `Src/analyzers/partition.py`、`graph_metrics.py`、`call_sequence.py` | 未移植 | 順序、scope/depth、SCC/循環、統計を個別に比較 |

## 現時点のCall graph契約

- 関数とメソッドをnodeにし、呼び出し元はCommon IRの`parent`と`name`から構成します。
- `calls`に記録された呼び出しをdirect relationとして保持します。重複した呼び出しedgeは保持し、fan-in/outの数え上げでは同一caller/calleeを一件として扱います。
- 最大depthはrootからedgeを何本進めるかで数えます。depth 0ではroot nodeのみ、depth 1ではrootから出るedgeまでを含めます。
- 循環は単純な有向cycleを列挙し、lexically最小のnodeから始まる形に正規化します。
- parserで確認された構文上の呼び出しを扱います。Common IRがcalleeの意味的解決を保持しない場合、実行時dispatchやimport bindingを推測して確定扱いにしません。

この表は移植済み範囲を示すもので、未移植項目のIssue #195完了を意味しません。追加時は共有fixture、Python版との比較、意図した差をこの表へ記録します。

