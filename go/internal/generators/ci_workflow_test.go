package generators

import (
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [TestBuildCIWorkflowDiagramConnectsJobsByNeeds: 依存jobだけをedge化し、独立jobをtriggerへ接続する]
//	}
func TestBuildCIWorkflowDiagramConnectsJobsByNeeds(t *testing.T) {
	workflow := models.CIWorkflow{
		Triggers: []string{"push"},
		Jobs: []models.CIJob{
			{ID: "build", Name: "Build", Steps: []string{"Compile"}},
			{ID: "test", Name: "Test", Needs: []string{"build", "build"}},
			{ID: "lint", Name: "Lint", Needs: []string{"missing"}},
		},
	}
	diagram := BuildCIWorkflowDiagram(workflow)
	if len(diagram.Triggers) != 1 || diagram.Triggers[0] != "push" || len(diagram.Jobs) != 3 {
		t.Fatalf("diagram = %#v", diagram)
	}
	if diagram.Jobs[0].Label != "Build\nCompile" {
		t.Fatalf("first job label = %q", diagram.Jobs[0].Label)
	}
	wantEdges := []CIJobEdge{{Target: "build", FromTrigger: true}, {Source: "build", Target: "test"}, {Target: "lint", FromTrigger: true}}
	if !reflect.DeepEqual(diagram.Edges, wantEdges) {
		t.Fatalf("edges = %#v, want %#v", diagram.Edges, wantEdges)
	}
}

//	{
//	  責務: [TestBuildCIWorkflowDiagramUsesFallbackWhenTriggersAreMissing: triggerがないworkflowにfallback表示とtrigger edgeを設定する]
//	}
func TestBuildCIWorkflowDiagramUsesFallbackWhenTriggersAreMissingOrBlank(t *testing.T) {
	cases := []struct {
		name     string
		triggers []string
	}{
		{name: "missing"},
		{name: "blank entries", triggers: []string{"", "   "}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			diagram := BuildCIWorkflowDiagram(models.CIWorkflow{
				Triggers: testCase.triggers,
				Jobs:     []models.CIJob{{ID: "build"}},
			})
			if len(diagram.Triggers) != 1 || diagram.Triggers[0] != "workflow" {
				t.Fatalf("triggers = %#v, want [workflow]", diagram.Triggers)
			}
			if len(diagram.Edges) != 1 || !diagram.Edges[0].FromTrigger || diagram.Edges[0].Target != "build" {
				t.Fatalf("edges = %#v, want one workflow-trigger edge to build", diagram.Edges)
			}
		})
	}
}
