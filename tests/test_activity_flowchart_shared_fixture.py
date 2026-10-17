import json
from pathlib import Path

from Src.models.activity import ActivityEdge, ActivityFlow, ActivityNode
from Src.renderers.flowchart import render_flowchart


FIXTURE = Path(__file__).parent / "fixtures" / "generators" / "activity_flowchart_v1.json"


def test_python_activity_flowchart_matches_shared_fixture():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    value = fixture["logical_input"]
    flow = ActivityFlow(
        entry=value["entry"],
        nodes=tuple(ActivityNode(**node) for node in value["nodes"]),
        edges=tuple(ActivityEdge(**edge) for edge in value["edges"]),
    )
    assert render_flowchart(flow) == fixture["expected_mermaid"]
