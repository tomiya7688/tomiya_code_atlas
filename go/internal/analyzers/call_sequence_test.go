package analyzers

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const callSequenceFixturePath = "../../../tests/fixtures/analyzers/call_sequence_v1.json"

func TestCallSequenceResolutionAndGraphMatchSharedGolden(t *testing.T) {
	var fixture struct {
		CommonIR commonir.Module `json:"common_ir"`
		Expected struct {
			Resolved   map[string][]ResolvedCall `json:"resolved"`
			GraphNodes []string                  `json:"graph_nodes"`
			GraphEdges []RelationEdge            `json:"graph_edges"`
		} `json:"expected"`
	}
	data, err := os.ReadFile(callSequenceFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	resolved := ResolveCallSequences(fixture.CommonIR)
	if !reflect.DeepEqual(resolved, fixture.Expected.Resolved) {
		t.Fatalf("resolved calls = %#v, want %#v", resolved, fixture.Expected.Resolved)
	}
	graph := BuildSequenceRelationGraph(fixture.CommonIR, resolved)
	if !reflect.DeepEqual(graph.Nodes(), fixture.Expected.GraphNodes) {
		t.Fatalf("nodes = %#v", graph.Nodes())
	}
	if !reflect.DeepEqual(graph.Edges(), fixture.Expected.GraphEdges) {
		t.Fatalf("edges = %#v", graph.Edges())
	}
}

