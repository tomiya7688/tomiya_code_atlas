package renderers

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

func TestActivityFlowchartRendererMatchesSharedFixture(t *testing.T) {
	fixtureBytes, err := os.ReadFile("../../../tests/fixtures/generators/activity_flowchart_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Flow            generators.ActivityFlow `json:"logical_input"`
		ExpectedMermaid string                  `json:"expected_mermaid"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	actual := RenderActivityFlowchart(generators.GenerateActivityFlowchart(fixture.Flow), "")
	if actual != fixture.ExpectedMermaid {
		t.Fatalf("activity flowchart mismatch\nactual:   %q\nexpected: %q", actual, fixture.ExpectedMermaid)
	}
}
