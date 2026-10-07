import json
from pathlib import Path

from Src.analyzers.ir import (
    ModuleIR,
    StateMachineIR,
    StateTransitionIR,
    TimingEventIR,
    TimingFlowIR,
)
from Src.generators.state_diagram import build_state_diagram_bundle
from Src.generators.timing_chart import build_timing_chart_bundle


FIXTURE = Path(__file__).parent / "fixtures" / "analyzers" / "state_timing_v1.json"


def test_python_state_and_timing_outputs_match_go_shared_golden():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    common_ir = fixture["common_ir"]
    machines = [
        StateMachineIR(
            owner=item["owner"],
            state_type=item["state_type"],
            state_variable=item["state_variable"],
            states=tuple(item["states"]),
            transitions=tuple(
                StateTransitionIR(
                    source=transition["source"],
                    target=transition["target"],
                    line=transition["line"],
                    event=transition.get("event"),
                    condition=transition.get("condition"),
                )
                for transition in item.get("transitions", [])
            ),
            initial_state=item.get("initial_state"),
            terminal_states=tuple(item.get("terminal_states", [])),
        )
        for item in common_ir["state_machines"]
    ]
    flows = [
        TimingFlowIR(
            owner=item["owner"],
            is_async=item["is_async"],
            events=tuple(
                TimingEventIR(
                    order=event["order"], kind=event["kind"], line=event["line"],
                    target=event.get("target"), detail=event.get("detail"),
                )
                for event in item["events"]
            ),
        )
        for item in common_ir["timing_flows"]
    ]
    module = ModuleIR(language=common_ir["language"], state_machines=machines, timing_flows=flows)
    expected = fixture["expected"]

    state = build_state_diagram_bundle(module)
    assert [
        {
            "name": diagram.name, "owner": diagram.owner,
            "state_type": diagram.state_type, "state_variable": diagram.state_variable,
            "states": list(diagram.states),
            "transitions": [
                {key: value for key, value in {
                    "source": transition.source, "target": transition.target,
                    "event": transition.event, "condition": transition.condition,
                }.items() if value is not None}
                for transition in diagram.transitions
            ],
            "initial_state": diagram.initial_state,
            "terminal_states": list(diagram.terminal_states),
        }
        for diagram in state.diagrams
    ] == expected["state"]["diagrams"]
    assert state.statistics == expected["state"]["statistics"]

    timing = build_timing_chart_bundle(module)
    assert [
        {
            "name": chart.name, "owner": chart.owner, "is_async": chart.is_async,
            "events": [
                {key: value for key, value in {
                    "order": event.order, "kind": event.kind, "line": event.line,
                    "target": event.target, "detail": event.detail,
                }.items() if value is not None}
                for event in chart.events
            ],
        }
        for chart in timing.charts
    ] == expected["timing"]["charts"]
    assert timing.statistics == expected["timing"]["statistics"]

