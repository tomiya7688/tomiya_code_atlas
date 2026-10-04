import json
from pathlib import Path

from Src.analyzers.package_dependencies import (
    ModuleDependencyUnit,
    build_package_dependency_graph,
)


FIXTURE = Path(__file__).parent / "fixtures" / "analyzers" / "package_dependencies_v1.json"


def test_python_package_analyzer_matches_shared_migration_golden():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    graph = build_package_dependency_graph(
        [
            ModuleDependencyUnit(module["name"], tuple(module["imports"]), module["is_package"])
            for module in fixture["modules"]
        ]
    )
    expected = fixture["expected"]

    assert sorted(graph.nodes) == expected["nodes"]
    assert sorted((edge.caller, edge.callee) for edge in graph.edges) == sorted(
        (edge["caller"], edge["callee"]) for edge in expected["edges"]
    )
    assert list(graph.isolated_nodes()) == expected["isolated_nodes"]
    assert [list(cycle) for cycle in graph.cycles()] == expected["cycles"]

