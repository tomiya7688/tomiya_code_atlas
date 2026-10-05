import json
from pathlib import Path

from Src.analyzers.class_relations import build_class_relation_graph
from Src.analyzers.component_dependencies import (
    ComponentDependencyUnit,
    build_component_dependency_graph,
)
from Src.analyzers.ir import CodeEntity, EntityKind, ModuleIR, ObjectInstanceIR
from Src.analyzers.object_relations import build_object_relation_graph
from Src.analyzers.package_dependencies import ModuleDependencyUnit


FIXTURE = Path(__file__).parent / "fixtures" / "analyzers" / "relations_v1.json"


def test_python_relation_analyzers_match_shared_go_migration_golden():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    common_ir = fixture["common_ir"]
    entities = [
        CodeEntity(
            kind=EntityKind(item["kind"]),
            name=item["name"],
            line=item.get("line", 1),
            end_line=item.get("end_line", 1),
            parent=item.get("parent"),
            bases=tuple(item.get("bases", ())),
            calls=tuple(item.get("calls", ())),
        )
        for item in common_ir["entities"]
    ]
    objects = [
        ObjectInstanceIR(
            name=item["name"],
            type_name=item["type_name"],
            line=item["line"],
            scope=item.get("scope"),
            references=tuple(tuple(ref) for ref in item.get("references", ())),
        )
        for item in common_ir["objects"]
    ]
    module = ModuleIR(language="python", entities=entities, objects=objects)
    expected = fixture["expected"]

    classes = build_class_relation_graph(module)
    assert sorted(classes.nodes) == expected["class_nodes"]
    assert [
        {"caller": edge.caller, "callee": edge.callee, "relation": edge.relation}
        for edge in classes.edges
    ] == expected["class_edges"]

    object_graph = build_object_relation_graph(module)
    assert sorted(object_graph.nodes) == expected["object_nodes"]
    assert [
        {"caller": edge.caller, "callee": edge.callee, "label": edge.label}
        for edge in object_graph.edges
    ] == expected["object_edges"]
    assert [list(cycle) for cycle in object_graph.cycles()] == expected["object_cycles"]

    components = build_component_dependency_graph(
        [
            ComponentDependencyUnit(
                ModuleDependencyUnit(item["name"], tuple(item["imports"])),
                item["component"],
            )
            for item in fixture["component_modules"]
        ]
    )
    assert sorted(components.nodes) == expected["component_nodes"]
    assert [
        {"caller": edge.caller, "callee": edge.callee}
        for edge in components.edges
    ] == expected["component_edges"]
    assert sorted(components.external_nodes) == expected["component_external_nodes"]
    assert [list(cycle) for cycle in components.cycles()] == expected["component_cycles"]

