package renderers

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

const rendererFixturePath = "../../../tests/fixtures/generators/renderers_v1.json"

//	{
//	  責務: [TestBoundaryDiagramReferencesAreVisible: partition boundaryのstubに参照先diagram名を描画する]
//	}
func TestBoundaryDiagramReferencesAreVisible(t *testing.T) {
	const nextDiagram = "shared_logger"
	callGraph := generators.CallGraph{Nodes: []string{"Logger"}, References: []generators.DiagramReference{{Node: "Logger", OutputName: nextDiagram}}}
	sequenceDiagram := generators.SequenceDiagram{Participants: []string{"Logger"}, References: []generators.DiagramReference{{Node: "Logger", OutputName: nextDiagram}}}
	communicationDiagram := generators.CommunicationDiagram{Participants: []string{"Logger"}, References: []generators.DiagramReference{{Node: "Logger", OutputName: nextDiagram}}}
	classDiagram := generators.ClassDiagram{Nodes: []generators.ClassNode{{Name: "Logger", External: true, ReferenceOutput: nextDiagram}}}
	objectDiagram := generators.ObjectDiagram{Nodes: []generators.ObjectNode{{Key: "logger", Name: "Logger", TypeName: "Logger", External: true, ReferenceOutput: nextDiagram}}}
	packageDiagram := generators.PackageDiagram{Nodes: []string{"shared.logger"}, ExternalNodes: []string{"shared.logger"}, References: []generators.DiagramReference{{Node: "shared.logger", OutputName: nextDiagram}}}
	componentDiagram := generators.ComponentDiagram{Nodes: []generators.ComponentDiagramNode{{Name: "Logger", Boundary: true, ReferenceOutput: nextDiagram}}}
	deploymentDiagram := generators.DeploymentDiagram{Nodes: []generators.DeploymentNode{{ID: "logger", Label: "Logger", Boundary: true, ReferenceOutput: nextDiagram}}}
	outputs := []string{
		RenderCallGraphMermaid(callGraph, "LR"),
		RenderSequenceDiagramMermaid(sequenceDiagram),
		RenderSequenceDiagramPlantUML(sequenceDiagram),
		RenderCommunicationDiagramMermaid(communicationDiagram),
		RenderClassDiagramMermaid(classDiagram),
		RenderClassDiagramPlantUML(classDiagram),
		RenderObjectDiagramMermaid(objectDiagram),
		RenderPackageDiagramMermaid(packageDiagram),
		RenderComponentDiagramMermaid(componentDiagram),
		RenderDeploymentDiagramMermaid(deploymentDiagram),
	}
	for index, output := range outputs {
		if !strings.Contains(output, nextDiagram) || !strings.Contains(output, "続き") {
			t.Errorf("diagram output %d omitted boundary destination: %s", index, output)
		}
	}
}

func stringPointer(value string) *string { return &value }

//	{
//	  責務: [TestSequenceRenderersPreserveMessageKinds: call、async、return、recursion、depth noteをMermaidとPlantUMLで区別する]
//	}
func TestSequenceRenderersPreserveMessageKinds(t *testing.T) {
	diagram := generators.SequenceDiagram{
		Name:         "interaction",
		Participants: []string{"alpha", "beta"},
		Messages: []generators.SequenceMessage{
			{Caller: "alpha", Callee: "beta", Label: "call", Kind: generators.SequenceCallMessage},
			{Caller: "alpha", Callee: "beta", Label: "async", Kind: generators.SequenceAsyncCallMessage},
			{Caller: "beta", Callee: "alpha", Label: "return", Kind: generators.SequenceReturnMessage},
			{Caller: "beta", Callee: "alpha", Label: "recursive", Kind: generators.SequenceRecursiveCall},
			{Caller: "alpha", Callee: "gamma", Label: "external", Kind: generators.SequenceCallMessage},
			{Caller: "beta", Callee: "next", Label: "Further calls omitted: next", Kind: generators.SequenceDepthTruncated},
		},
	}

	mermaid := RenderSequenceDiagramMermaid(diagram)
	for _, expected := range []string{"p0->>p1: call", "p0-)p1: async", "p1-->>p0: return", "p1->>p0: [recursive] recursive", "participant p2 as gamma", "p0->>p2: external", "Note over p1: Further calls omitted- next"} {
		if !strings.Contains(mermaid, expected) {
			t.Errorf("Mermaid output does not contain %q:\n%s", expected, mermaid)
		}
	}
	if strings.Contains(mermaid, "participant p2 as next") {
		t.Errorf("Mermaid output declared an uncalled truncated target:\n%s", mermaid)
	}

	plantUML := RenderSequenceDiagramPlantUML(diagram)
	alpha := hashedID("p_", "alpha")
	beta := hashedID("p_", "beta")
	for _, expected := range []string{
		alpha + " -> " + beta + " : call",
		alpha + " ->> " + beta + " : async",
		beta + " --> " + alpha + " : return",
		beta + " -> " + alpha + " : [recursive] recursive",
		alpha + " -> " + hashedID("p_", "gamma") + " : external",
		"note over " + beta + " : Further calls omitted: next",
	} {
		if !strings.Contains(plantUML, expected) {
			t.Errorf("PlantUML output does not contain %q:\n%s", expected, plantUML)
		}
	}
}

//	{
//	  責務: [TestCommunicationRendererLabelsRecursionAndTruncation: communication diagramでrecursionと省略理由を明示する]
//	}
func TestCommunicationRendererLabelsRecursionAndTruncation(t *testing.T) {
	diagram := generators.CommunicationDiagram{
		Name:         "interaction",
		Participants: []string{"alpha", "beta"},
		Messages: []generators.CommunicationMessage{
			{Source: "alpha", Target: "beta", Order: "1", Label: "call", Kind: generators.SequenceRecursiveCall},
			{Source: "beta", Target: "gamma", Order: "2", Label: "Further calls omitted: leaf", Kind: generators.SequenceDepthTruncated},
		},
	}

	output := RenderCommunicationDiagramMermaid(diagram)
	for _, expected := range []string{"p2[\"gamma\"]", "1: [recursive] call", "2: [depth truncated] Further calls omitted: leaf"} {
		if !strings.Contains(output, expected) {
			t.Errorf("communication output does not contain %q:\n%s", expected, output)
		}
	}
}

//	{
//	  責務: [TestSequenceRenderersSkipMessagesWithoutRequiredParticipants: 空participantを含む不正messageを出力しない]
//	}
func TestSequenceRenderersSkipMessagesWithoutRequiredParticipants(t *testing.T) {
	diagram := generators.SequenceDiagram{
		Messages: []generators.SequenceMessage{
			{Caller: "", Callee: "beta", Label: "missing caller", Kind: generators.SequenceCallMessage},
			{Caller: "alpha", Callee: "", Label: "missing callee", Kind: generators.SequenceCallMessage},
			{Caller: "", Callee: "alpha", Label: "missing truncation caller", Kind: generators.SequenceDepthTruncated},
			{Caller: "alpha", Callee: "beta", Label: "unknown kind", Kind: "future-kind"},
		},
	}
	for name, output := range map[string]string{
		"Mermaid":  RenderSequenceDiagramMermaid(diagram),
		"PlantUML": RenderSequenceDiagramPlantUML(diagram),
	} {
		if strings.Contains(output, "missing caller") || strings.Contains(output, "missing callee") || strings.Contains(output, "missing truncation caller") || strings.Contains(output, "unknown kind") {
			t.Errorf("%s contains a message with an empty participant:\n%s", name, output)
		}
		if strings.Contains(output, "Note over :") || strings.Contains(output, "note over  :") {
			t.Errorf("%s contains a truncation note without a caller:\n%s", name, output)
		}
	}
}

//	{
//	  責務: [TestRenderersMatchSharedGolden: Pythonと共有するgoldenで全1.0 rendererの改行・順序・escapingを固定する]
//	}
func TestRenderersMatchSharedGolden(t *testing.T) {
	var fixture struct {
		Expected map[string]string `json:"expected"`
	}
	data, err := os.ReadFile(rendererFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	classDiagram := generators.ClassDiagram{Name: "sample", Nodes: []generators.ClassNode{{Name: "Service", Members: []generators.ClassMember{{Name: "run", Parameters: []string{"x"}, Visibility: "public"}}}, {Name: "Repo"}}, Relations: []generators.ClassRelation{{Source: "Service", Target: "Repo", Kind: "uses"}}}
	sequenceDiagram := generators.SequenceDiagram{Name: "sample", Participants: []string{"Service.run", "Repo.load"}, Messages: []generators.SequenceMessage{{Caller: "Service.run", Callee: "Repo.load", Label: "load: item", Kind: "call"}, {Caller: "Repo.load", Callee: "Service.run", Label: "return", Kind: "return"}}}
	communicationDiagram := generators.CommunicationDiagram{Name: "sample", Participants: []string{"Service", "Repo"}, Messages: []generators.CommunicationMessage{{Source: "Service", Target: "Repo", Order: "1", Label: "load item"}}}
	objectDiagram := generators.ObjectDiagram{Name: "sample", Nodes: []generators.ObjectNode{{Key: "build:service", Name: "service", TypeName: "Service", Fields: []generators.ObjectField{{Name: "name", Value: "api"}}}, {Key: "build:repo", Name: "repo", TypeName: "Repo"}}, References: []generators.ObjectReference{{Source: "build:service", Target: "build:repo", Label: "repo"}}}
	packageDiagram := generators.PackageDiagram{Name: "sample", Nodes: []string{"app", "data"}, Edges: []generators.PackageDiagramEdge{{Source: "app", Target: "data"}}, CycleNodes: []string{"app", "data"}, IsolatedNodes: []string{"data"}}
	componentDiagram := generators.ComponentDiagram{Name: "sample", Nodes: []generators.ComponentDiagramNode{{Name: "app"}, {Name: "external:requests", External: true}}, Edges: []generators.ComponentDiagramEdge{{Source: "app", Target: "external:requests"}}}
	initial, event, condition := "IDLE", "start", "ready"
	stateDiagram := generators.StateDiagram{Name: "sample", Owner: "Worker", StateType: "State", StateVariable: "state", States: []string{"IDLE", "RUNNING"}, Transitions: []generators.StateDiagramTransition{{Source: "*", Target: "RUNNING", Event: &event, Condition: &condition}}, InitialState: &initial, TerminalStates: []string{"RUNNING"}}
	timingChart := generators.TimingChart{Name: "sample", Owner: "Worker.load", IsAsync: true, Events: []generators.TimingChartEvent{{Order: 1, Kind: "await", Target: stringPointer("fetch"), Detail: stringPointer("timeout=1")}, {Order: 2, Kind: "parallel_start"}}}
	useCaseDiagram := generators.UseCaseDiagram{Name: "sample", Scope: "Controller", UseCases: []generators.UseCase{{Name: "Open", Actor: "User", Component: "button", Event: "click", Handler: "Controller.on_open", CallChain: []string{"Repo.load"}}}}
	deploymentDiagram := generators.DeploymentDiagram{Name: "sample", Nodes: []generators.DeploymentNode{{ID: "api", Label: "API", Kind: "workload", Confidence: "confirmed", Metadata: map[string]string{"image": "app:v1"}}, {ID: "db", Label: "DB", Kind: "database", Confidence: "inferred", Environment: "prod"}}, Connections: []generators.DeploymentConnection{{Source: "api", Target: "db", Relation: "connects", Confidence: "inferred"}}}
	callGraph := generators.CallGraph{Nodes: []string{"main", "helper"}, Edges: []generators.CallGraphEdge{{Caller: "main", Callee: "helper"}}}
	rows := []generators.ResponsibilityRow{{ClassName: "Service", Responsibility: "Coordinates service processing."}}

	got := map[string]string{
		"call_graph_mermaid":      RenderCallGraphMermaid(callGraph, "LR"),
		"class_mermaid":           RenderClassDiagramMermaid(classDiagram),
		"class_plantuml":          RenderClassDiagramPlantUML(classDiagram),
		"sequence_mermaid":        RenderSequenceDiagramMermaid(sequenceDiagram),
		"sequence_plantuml":       RenderSequenceDiagramPlantUML(sequenceDiagram),
		"communication_mermaid":   RenderCommunicationDiagramMermaid(communicationDiagram),
		"object_mermaid":          RenderObjectDiagramMermaid(objectDiagram),
		"package_mermaid":         RenderPackageDiagramMermaid(packageDiagram),
		"component_mermaid":       RenderComponentDiagramMermaid(componentDiagram),
		"state_mermaid":           RenderStateDiagramMermaid(stateDiagram),
		"timing_mermaid":          RenderTimingChartMermaid(timingChart),
		"use_case_mermaid":        RenderUseCaseDiagramMermaid(useCaseDiagram),
		"deployment_mermaid":      RenderDeploymentDiagramMermaid(deploymentDiagram),
		"responsibility_markdown": RenderResponsibilityMarkdown(rows),
		"responsibility_csv":      RenderResponsibilityCSV(rows),
	}
	for name, expected := range fixture.Expected {
		if got[name] != expected {
			t.Errorf("%s mismatch\n--- got ---\n%s--- want ---\n%s", name, got[name], expected)
		}
	}
}
