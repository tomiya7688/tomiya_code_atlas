import json
from pathlib import Path

from Src.analyzers.component_dependencies import (
    ComponentDependencyUnit,
    build_component_dependency_graph,
)
from Src.analyzers.deployment import merge_topologies, topology_from_component_graph
from Src.analyzers.package_dependencies import ModuleDependencyUnit
from Src.models.deployment import (
    DeploymentConnection,
    DeploymentNode,
    DeploymentTopology,
)


FIXTURE = Path(__file__).parent / "fixtures" / "analyzers" / "deployment_topology_v1.json"


def _topology(value):
    return DeploymentTopology(
        nodes=tuple(DeploymentNode(**item) for item in value["nodes"]),
        connections=tuple(DeploymentConnection(**item) for item in value["connections"]),
    )


def _node_value(node):
    value = {"id": node.id, "label": node.label, "kind": node.kind, "confidence": node.confidence}
    if node.environment:
        value["environment"] = node.environment
    if node.source:
        value["source"] = node.source
    if node.metadata:
        value["metadata"] = dict(node.metadata)
    return value


def test_python_deployment_topology_matches_go_migration_golden():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    merged = merge_topologies(*(_topology(value) for value in fixture["merge"]["topologies"]))
    assert [_node_value(node) for node in merged.nodes] == fixture["merge"]["expected"]["nodes"]
    assert [
        {"source": edge.source, "target": edge.target, "relation": edge.relation, "confidence": edge.confidence}
        for edge in merged.connections
    ] == fixture["merge"]["expected"]["connections"]

    units = [
        ComponentDependencyUnit(
            ModuleDependencyUnit(item["name"], tuple(item["imports"])), item["component"]
        )
        for item in fixture["component_modules"]
    ]
    graph = build_component_dependency_graph(units)
    topology = topology_from_component_graph(graph)
    assert [
        _node_value(node) for node in topology.nodes
    ] == fixture["expected_component_topology"]["nodes"]
    assert [
        {"source": edge.source, "target": edge.target,
         "relation": edge.relation, "confidence": edge.confidence}
        for edge in topology.connections
    ] == fixture["expected_component_topology"]["connections"]

