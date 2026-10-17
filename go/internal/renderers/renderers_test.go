package renderers

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

const rendererFixturePath = "../../../tests/fixtures/generators/renderers_v1.json"

func stringPointer(value string) *string { return &value }

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
