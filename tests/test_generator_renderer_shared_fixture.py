import json
from pathlib import Path

from Src.analyzers.call_graph import CallEdge, CallGraph
from Src.generators.component_diagram import ComponentDiagram, ComponentDiagramEdge, ComponentDiagramNode
from Src.generators.deployment_diagram import DeploymentDiagram
from Src.generators.package_diagram import PackageDiagram, PackageDiagramEdge
from Src.generators.state_diagram import StateDiagram, StateDiagramTransition
from Src.generators.timing_chart import TimingChart, TimingChartEvent
from Src.generators.use_case_diagram import UseCase, UseCaseDiagram
from Src.models.class_diagram import ClassDiagram, ClassMember, ClassNode, ClassRelation
from Src.models.communication_diagram import CommunicationDiagram, CommunicationMessage
from Src.models.deployment import DeploymentConnection, DeploymentNode
from Src.models.object_diagram import ObjectDiagram, ObjectField, ObjectNode, ObjectReference
from Src.models.responsibility import ResponsibilityRow
from Src.models.sequence_diagram import SequenceDiagram, SequenceMessage
from Src.renderers.mermaid_call_graph import render_call_graph
from Src.renderers.mermaid_class_diagram import render_class_diagram
from Src.renderers.mermaid_communication_diagram import render_communication_diagram
from Src.renderers.mermaid_component_diagram import render_component_diagram
from Src.renderers.mermaid_deployment_diagram import render_deployment_diagram
from Src.renderers.mermaid_object_diagram import render_object_diagram
from Src.renderers.mermaid_package_diagram import render_package_diagram
from Src.renderers.mermaid_sequence_diagram import render_sequence_diagram
from Src.renderers.mermaid_state_diagram import render_state_diagram
from Src.renderers.mermaid_timing_chart import render_timing_chart
from Src.renderers.mermaid_use_case_diagram import render_use_case_diagram
from Src.renderers.plantuml_class_diagram import render_class_diagram as render_class_plantuml
from Src.renderers.plantuml_sequence_diagram import render_sequence_diagram as render_sequence_plantuml
from Src.renderers.responsibility_table import render_responsibility_csv, render_responsibility_markdown

FIXTURE = Path(__file__).parent / "fixtures" / "generators" / "renderers_v1.json"


def test_python_renderers_match_go_shared_golden():
    expected = json.loads(FIXTURE.read_text(encoding="utf-8"))["expected"]

    class_diagram = ClassDiagram(
        "sample",
        (
            ClassNode("Service", (ClassMember("run", ("x",), "public"),)),
            ClassNode("Repo"),
        ),
        (ClassRelation("Service", "Repo", "uses"),),
    )
    sequence_diagram = SequenceDiagram(
        "sample",
        ("Service.run", "Repo.load"),
        (
            SequenceMessage("Service.run", "Repo.load", "load: item", kind="call"),
            SequenceMessage("Repo.load", "Service.run", "return", kind="return"),
        ),
    )
    communication_diagram = CommunicationDiagram(
        "sample",
        ("Service", "Repo"),
        (CommunicationMessage("Service", "Repo", "1", "load item"),),
    )
    object_diagram = ObjectDiagram(
        "sample",
        (
            ObjectNode("build:service", "service", "Service", (ObjectField("name", "api"),)),
            ObjectNode("build:repo", "repo", "Repo"),
        ),
        (ObjectReference("build:service", "build:repo", "repo"),),
    )
    package_diagram = PackageDiagram(
        "sample",
        ("app", "data"),
        (PackageDiagramEdge("app", "data"),),
        ("app", "data"),
        ("data",),
    )
    component_diagram = ComponentDiagram(
        "sample",
        (ComponentDiagramNode("app"), ComponentDiagramNode("external:requests", True)),
        (ComponentDiagramEdge("app", "external:requests"),),
    )
    state_diagram = StateDiagram(
        "sample",
        "Worker",
        "State",
        "state",
        ("IDLE", "RUNNING"),
        (StateDiagramTransition("*", "RUNNING", "start", "ready"),),
        "IDLE",
        ("RUNNING",),
    )
    timing_chart = TimingChart(
        "sample",
        "Worker.load",
        True,
        (
            TimingChartEvent(1, "await", 0, "fetch", "timeout=1"),
            TimingChartEvent(2, "parallel_start", 0),
        ),
    )
    use_case_diagram = UseCaseDiagram(
        "sample",
        "Controller",
        (UseCase("Open", "User", "button", "click", "Controller.on_open", ("Repo.load",)),),
    )
    deployment_diagram = DeploymentDiagram(
        "sample",
        (
            DeploymentNode("api", "API", "workload", "confirmed", metadata=(("image", "app:v1"),)),
            DeploymentNode("db", "DB", "database", "inferred", environment="prod"),
        ),
        (DeploymentConnection("api", "db", "connects", "inferred"),),
    )
    call_graph = CallGraph([CallEdge("main", "helper")], explicit_nodes={"main", "helper"})
    rows = (ResponsibilityRow("Service", "Coordinates service processing."),)

    actual = {
        "call_graph_mermaid": render_call_graph(call_graph),
        "class_mermaid": render_class_diagram(class_diagram),
        "class_plantuml": render_class_plantuml(class_diagram),
        "sequence_mermaid": render_sequence_diagram(sequence_diagram),
        "sequence_plantuml": render_sequence_plantuml(sequence_diagram),
        "communication_mermaid": render_communication_diagram(communication_diagram),
        "object_mermaid": render_object_diagram(object_diagram),
        "package_mermaid": render_package_diagram(package_diagram),
        "component_mermaid": render_component_diagram(component_diagram),
        "state_mermaid": render_state_diagram(state_diagram),
        "timing_mermaid": render_timing_chart(timing_chart),
        "use_case_mermaid": render_use_case_diagram(use_case_diagram),
        "deployment_mermaid": render_deployment_diagram(deployment_diagram),
        "responsibility_markdown": render_responsibility_markdown(rows),
        "responsibility_csv": render_responsibility_csv(rows),
    }

    assert actual == expected
