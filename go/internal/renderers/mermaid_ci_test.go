package renderers

import (
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

//	{
//	  責務: [TestRenderCIWorkflowMermaidEscapesLabelsAndEdges: CI図のjob labelをescapeし依存関係をMermaidへ直列化する]
//	}
func TestRenderCIWorkflowMermaidEscapesLabelsAndEdges(t *testing.T) {
	diagram := generators.CIWorkflowDiagram{
		Triggers: []string{"push\"", "pull_request"},
		Jobs: []generators.CIJobNode{
			{ID: "build-job", Label: "Build\nRun <tests>"},
			{ID: "build_job", Label: "Build job"},
			{ID: "trigger", Label: "Trigger job"},
		},
		Edges: []generators.CIJobEdge{
			{Target: "build-job", FromTrigger: true},
			{Source: "trigger", Target: "build_job"},
		},
	}
	output := RenderCIWorkflowMermaid(diagram)
	for _, expected := range []string{`ci_trigger["push&quot;, pull_request"]`, `ci_job_6275696c642d6a6f62["Build<br/>Run &lt;tests&gt;"]`, `ci_job_6275696c645f6a6f62["Build job"]`, `ci_job_74726967676572["Trigger job"]`, "ci_trigger --> ci_job_6275696c642d6a6f62", "ci_job_74726967676572 --> ci_job_6275696c645f6a6f62"} {
		if !strings.Contains(output, expected) {
			t.Errorf("rendered CI graph does not contain %q: %q", expected, output)
		}
	}
}
