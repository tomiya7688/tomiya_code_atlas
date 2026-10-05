import json
from pathlib import Path

from Src.analyzers.call_sequence import (
    build_sequence_relation_graph,
    resolve_call_sequences,
)
from Src.analyzers.ir import CodeEntity, EntityKind, ModuleIR


FIXTURE = Path(__file__).parent / "fixtures" / "analyzers" / "call_sequence_v1.json"


def test_python_call_sequence_matches_shared_go_golden():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    entities = [
        CodeEntity(
            kind=EntityKind(item["kind"]),
            name=item["name"],
            line=item["line"],
            end_line=item["end_line"],
            parent=item.get("parent"),
            calls=tuple(item.get("calls", ())),
            call_sequence=tuple(item.get("call_sequence", ())),
        )
        for item in fixture["common_ir"]["entities"]
    ]
    module = ModuleIR(language="python", entities=entities)
    resolved = resolve_call_sequences(module)
    expected = fixture["expected"]
    actual_resolved = {
        owner: [{"raw": call.raw, "target": call.target} for call in calls]
        for owner, calls in resolved.items()
    }
    assert actual_resolved == expected["resolved"]

    graph = build_sequence_relation_graph(module, resolved)
    assert sorted(graph.nodes) == expected["graph_nodes"]
    assert [
        {"caller": edge.caller, "callee": edge.callee}
        for edge in graph.edges
    ] == expected["graph_edges"]

