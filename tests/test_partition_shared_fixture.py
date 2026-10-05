import json
from pathlib import Path

from Src.analyzers.call_graph import CallEdge, CallGraph
from Src.analyzers.graph_metrics import (
    cyclic_strongly_connected_components,
    strongly_connected_components,
)
from Src.analyzers.partition import partition_graph


FIXTURE = Path(__file__).parent / "fixtures" / "analyzers" / "partition_v1.json"


class _GraphWithExplicitNodes:
    def __init__(self, nodes, edges):
        self._graph = CallGraph(edges)
        self.nodes = set(nodes) | self._graph.nodes
        self.edges = self._graph.edges

    def fan_in(self):
        values = self._graph.fan_in()
        return {node: values.get(node, 0) for node in self.nodes}

    def fan_out(self):
        values = self._graph.fan_out()
        return {node: values.get(node, 0) for node in self.nodes}

    def cycles(self):
        return self._graph.cycles()


def _graph(case):
    edges = [CallEdge(*edge) for edge in case["edges"]]
    return _GraphWithExplicitNodes(case["nodes"], edges)


def test_python_partition_and_scc_match_shared_go_golden():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    for case in fixture["cases"]:
        result = partition_graph(
            _graph(case),
            fan_in_threshold=case["fan_in_threshold"],
            min_child_series_size=case["min_child_series_size"],
        )
        expected = case["expected"]
        assert [list(series) for series in result.series] == expected["series"], case["name"]
        assert list(result.shared) == expected["shared"], case["name"]
        assert list(result.series_roots) == expected["series_roots"], case["name"]
        assert list(result.series_depths) == expected["series_depths"], case["name"]
        assert list(result.series_parents) == expected["series_parents"], case["name"]
        assert result.cross_series_edge_count == expected["cross_series_edge_count"], case["name"]
        assert result.cycle_count == expected["cycle_count"], case["name"]
        assert list(result.fan_in_distribution) == expected["fan_in_distribution"], case["name"]
        assert list(result.fan_out_distribution) == expected["fan_out_distribution"], case["name"]

    metrics = fixture["metrics_graph"]
    graph = _graph(metrics)
    assert [list(item) for item in strongly_connected_components(graph)] == metrics["expected_scc"]
    assert [list(item) for item in cyclic_strongly_connected_components(graph)] == metrics["expected_cyclic_scc"]

