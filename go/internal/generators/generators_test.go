package generators

import (
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

func pointer(value string) *string { return &value }

//	{
//	  責務: [TestGeneratorInventoryBuildsRendererNeutralModels: 1.0のgenerator inventoryが共通IR/Analyzerからlogical modelを作れることを検証する]
//	}
func TestGeneratorInventoryBuildsRendererNeutralModels(t *testing.T) {
	module := commonir.Module{
		Language: "python",
		Entities: []commonir.Entity{
			{Kind: "class", Name: "Repo", SourceLocation: commonir.SourceLocation{Line: 1, EndLine: 1}},
			{Kind: "class", Name: "Service", SourceLocation: commonir.SourceLocation{Line: 2, EndLine: 4}},
			{Kind: "method", Name: "run", Parent: pointer("Service"), Calls: []string{"Repo", "helper"}, CallSequence: []string{"helper"}, Visibility: "public", SourceLocation: commonir.SourceLocation{Line: 3, EndLine: 3}},
			{Kind: "function", Name: "main", Calls: []string{"helper"}, CallSequence: []string{"helper"}, SourceLocation: commonir.SourceLocation{Line: 5, EndLine: 5}},
			{Kind: "function", Name: "helper", SourceLocation: commonir.SourceLocation{Line: 6, EndLine: 6}},
		},
		Objects: []commonir.ObjectInstance{
			{Name: "service", TypeName: "Service", Scope: pointer("build"), Values: [][]string{{"name", "api"}}, References: [][]string{{"repo", "repo"}}},
			{Name: "repo", TypeName: "Repo", Scope: pointer("build")},
		},
		InputEvents: []commonir.InputEvent{{Actor: "User", Component: "button", Event: "click", Handler: "main", Label: pointer("Run"), Line: 8}},
		Imports:     []string{},
		Diagnostics: []commonir.Diagnostic{},
	}

	callGraph, err := BuildCallGraph(module, nil, nil)
	if err != nil || len(callGraph.Edges) == 0 {
		t.Fatalf("call graph = %#v, err = %v", callGraph, err)
	}
	callBundle, err := BuildCallGraphBundle(module, 3, nil, nil)
	if err != nil || len(callBundle.Diagrams) == 0 || len(callBundle.Placements) != len(callBundle.Diagrams) {
		t.Fatalf("call bundle = %#v, err = %v", callBundle, err)
	}
	classBundle, err := BuildClassDiagramBundle(module, 3, nil)
	if err != nil || len(classBundle.Diagrams) == 0 {
		t.Fatalf("class bundle = %#v, err = %v", classBundle, err)
	}
	objectBundle, err := BuildObjectDiagramBundle(module, 3)
	if err != nil || len(objectBundle.Diagrams) == 0 {
		t.Fatalf("object bundle = %#v, err = %v", objectBundle, err)
	}
	responsibilityBundle, err := BuildResponsibilityTableBundle(module, 3)
	if err != nil || len(responsibilityBundle.Tables) == 0 {
		t.Fatalf("responsibility bundle = %#v, err = %v", responsibilityBundle, err)
	}
	sequenceBundle, err := BuildSequenceDiagramBundle(module, 3, 8, nil)
	if err != nil || len(sequenceBundle.Diagrams) == 0 {
		t.Fatalf("sequence bundle = %#v, err = %v", sequenceBundle, err)
	}
	communicationBundle, err := BuildCommunicationDiagramBundle(module, 3, 8, true)
	if err != nil || len(communicationBundle.Diagrams) != len(sequenceBundle.Diagrams) {
		t.Fatalf("communication bundle = %#v, err = %v", communicationBundle, err)
	}
	useCaseBundle, err := BuildUseCaseDiagramBundle(module, nil, 5)
	if err != nil || len(useCaseBundle.Diagrams) != 1 || useCaseBundle.Diagrams[0].UseCases[0].Name != "Run" {
		t.Fatalf("use case bundle = %#v, err = %v", useCaseBundle, err)
	}
}

func TestProjectAndDeploymentGeneratorsBuildFromAnalyzerResults(t *testing.T) {
	units := []analyzers.ModuleDependencyUnit{
		{Name: "app.main", CommonIR: commonir.Module{Language: "python", Entities: []commonir.Entity{}, Imports: []string{"data.repo", "requests"}, Diagnostics: []commonir.Diagnostic{}}},
		{Name: "data.repo", CommonIR: commonir.Module{Language: "python", Entities: []commonir.Entity{}, Imports: []string{}, Diagnostics: []commonir.Diagnostic{}}},
	}
	packageGraph := analyzers.BuildPackageDependencyGraph(units)
	packageBundle, err := BuildPackageDiagramBundle(packageGraph, 3)
	if err != nil || len(packageBundle.Diagrams) == 0 {
		t.Fatalf("package bundle = %#v, err = %v", packageBundle, err)
	}
	componentGraph := analyzers.BuildComponentDependencyGraph([]analyzers.ComponentDependencyUnit{{Module: units[0], Component: "app"}, {Module: units[1], Component: "data"}})
	componentBundle, err := BuildComponentDiagramBundle(componentGraph, 3)
	if err != nil || len(componentBundle.Diagrams) == 0 {
		t.Fatalf("component bundle = %#v, err = %v", componentBundle, err)
	}
	topology := analyzers.DeploymentTopology{
		Nodes:       []analyzers.DeploymentNode{{ID: "api", Label: "API", Kind: "workload", Confidence: analyzers.ConfidenceConfirmed}, {ID: "db", Label: "DB", Kind: "database", Confidence: analyzers.ConfidenceInferred}},
		Connections: []analyzers.DeploymentConnection{{Source: "api", Target: "db", Relation: "connects", Confidence: analyzers.ConfidenceInferred}},
	}
	deploymentBundle, err := BuildDeploymentDiagramBundle(topology, 3)
	if err != nil || len(deploymentBundle.Diagrams) == 0 || deploymentBundle.Statistics["confirmed_node_count"] != 1 {
		t.Fatalf("deployment bundle = %#v, err = %v", deploymentBundle, err)
	}
}

func TestGeneratorValidationRejectsInvalidBounds(t *testing.T) {
	negative := -1
	if _, err := BuildCallGraph(commonir.Module{}, nil, &negative); err == nil {
		t.Fatal("negative call graph depth must fail")
	}
	if _, err := BuildSequenceDiagramBundle(commonir.Module{}, 3, -1, nil); err == nil {
		t.Fatal("negative sequence depth must fail")
	}
	if _, err := BuildUseCaseDiagramBundle(commonir.Module{}, nil, -1); err == nil {
		t.Fatal("negative use case depth must fail")
	}
}
