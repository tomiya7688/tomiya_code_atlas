package analyzers

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

const partitionFixturePath = "../../../tests/fixtures/analyzers/partition_v1.json"

func TestPartitionAndGraphMetricsMatchSharedGolden(t *testing.T) {
	type expected struct {
		Series     [][]string `json:"series"`
		Shared     []string   `json:"shared"`
		Roots      []string   `json:"series_roots"`
		Depths     []int      `json:"series_depths"`
		Parents    []*string  `json:"series_parents"`
		CrossEdges int        `json:"cross_series_edge_count"`
		Cycles     int        `json:"cycle_count"`
		FanIn      []int      `json:"fan_in_distribution"`
		FanOut     []int      `json:"fan_out_distribution"`
	}
	type partitionCase struct {
		Name               string     `json:"name"`
		Nodes              []string   `json:"nodes"`
		Edges              [][]string `json:"edges"`
		FanInThreshold     int        `json:"fan_in_threshold"`
		MinChildSeriesSize int        `json:"min_child_series_size"`
		Expected           expected   `json:"expected"`
	}
	var fixture struct {
		Cases        []partitionCase `json:"cases"`
		MetricsGraph struct {
			Nodes             []string   `json:"nodes"`
			Edges             [][]string `json:"edges"`
			ExpectedSCC       [][]string `json:"expected_scc"`
			ExpectedCyclicSCC [][]string `json:"expected_cyclic_scc"`
		} `json:"metrics_graph"`
	}
	data, err := os.ReadFile(partitionFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			edges := make([]RelationEdge, 0, len(tc.Edges))
			for _, pair := range tc.Edges {
				edges = append(edges, RelationEdge{Caller: pair[0], Callee: pair[1]})
			}
			graph, err := NewRelationGraph(tc.Nodes, edges)
			if err != nil {
				t.Fatal(err)
			}
			partition, err := PartitionGraph(graph, tc.FanInThreshold, tc.MinChildSeriesSize)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.Expected
			if !reflect.DeepEqual(partition.Series, want.Series) || !reflect.DeepEqual(partition.Shared, want.Shared) || !reflect.DeepEqual(partition.SeriesRoots, want.Roots) || !reflect.DeepEqual(partition.SeriesDepths, want.Depths) || !reflect.DeepEqual(partition.SeriesParents, want.Parents) || partition.CrossSeriesEdgeCount != want.CrossEdges || partition.CycleCount != want.Cycles || !reflect.DeepEqual(partition.FanInDistribution, want.FanIn) || !reflect.DeepEqual(partition.FanOutDistribution, want.FanOut) {
				t.Fatalf("partition mismatch: got %#v, want %#v", partition, want)
			}
		})
	}
	metricEdges := make([]RelationEdge, 0, len(fixture.MetricsGraph.Edges))
	for _, pair := range fixture.MetricsGraph.Edges {
		metricEdges = append(metricEdges, RelationEdge{Caller: pair[0], Callee: pair[1]})
	}
	metrics, err := NewRelationGraph(fixture.MetricsGraph.Nodes, metricEdges)
	if err != nil {
		t.Fatal(err)
	}
	if got := StronglyConnectedComponents(metrics); !reflect.DeepEqual(got, fixture.MetricsGraph.ExpectedSCC) {
		t.Fatalf("SCCs = %#v", got)
	}
	if got := CyclicStronglyConnectedComponents(metrics); !reflect.DeepEqual(got, fixture.MetricsGraph.ExpectedCyclicSCC) {
		t.Fatalf("cyclic SCCs = %#v", got)
	}
}

