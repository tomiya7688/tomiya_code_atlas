package analyzers

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const sharedFixturePath = "../../../tests/fixtures/backend_conformance/serialized_common_ir_v1.json"
const packageDependencyFixturePath = "../../../tests/fixtures/analyzers/package_dependencies_v1.json"

func TestPackageDependenciesMatchSharedGolden(t *testing.T) {
	type fixtureModule struct {
		Name      string   `json:"name"`
		Imports   []string `json:"imports"`
		IsPackage bool     `json:"is_package"`
	}
	type expected struct {
		Nodes    []string                `json:"nodes"`
		Edges    []PackageDependencyEdge `json:"edges"`
		Isolated []string                `json:"isolated_nodes"`
		Cycles   [][]string              `json:"cycles"`
	}
	var fixture struct {
		Modules  []fixtureModule `json:"modules"`
		Expected expected        `json:"expected"`
	}
	data, err := os.ReadFile(packageDependencyFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	units := make([]ModuleDependencyUnit, 0, len(fixture.Modules))
	for _, module := range fixture.Modules {
		units = append(units, ModuleDependencyUnit{
			Name: module.Name, IsPackage: module.IsPackage,
			CommonIR: commonir.Module{Language: "python", Entities: []commonir.Entity{}, Imports: module.Imports, Diagnostics: []commonir.Diagnostic{}},
		})
	}
	graph := BuildPackageDependencyGraph(units)
	if !reflect.DeepEqual(graph.Nodes(), fixture.Expected.Nodes) {
		t.Fatalf("nodes = %#v", graph.Nodes())
	}
	if !reflect.DeepEqual(graph.Edges(), fixture.Expected.Edges) {
		t.Fatalf("edges = %#v", graph.Edges())
	}
	if !reflect.DeepEqual(graph.IsolatedNodes(), fixture.Expected.Isolated) {
		t.Fatalf("isolated = %#v", graph.IsolatedNodes())
	}
	cycles, err := graph.Cycles()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cycles, fixture.Expected.Cycles) {
		t.Fatalf("cycles = %#v", cycles)
	}
}

func TestCallGraphMatchesSharedPythonGolden(t *testing.T) {
	data, err := os.ReadFile(sharedFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := commonir.DecodeFixture(data)
	if err != nil {
		t.Fatalf("decode shared fixture: %v", err)
	}
	graph := BuildCallGraph(fixture.CommonIR)
	if !reflect.DeepEqual(graph.Nodes(), fixture.LogicalOutput.CallGraph.Nodes) {
		t.Fatalf("nodes = %#v, want %#v", graph.Nodes(), fixture.LogicalOutput.CallGraph.Nodes)
	}
	if !reflect.DeepEqual(graph.Edges(), fixture.LogicalOutput.CallGraph.Edges) {
		t.Fatalf("edges = %#v, want %#v", graph.Edges(), fixture.LogicalOutput.CallGraph.Edges)
	}
}

func TestCallGraphMetricsReachabilityAndCycles(t *testing.T) {
	module := commonir.Module{Entities: []commonir.Entity{
		{Kind: "function", Name: "a", Calls: []string{"b", "b"}},
		{Kind: "function", Name: "b", Calls: []string{"c"}},
		{Kind: "function", Name: "c", Calls: []string{"a", "d"}},
		{Kind: "function", Name: "d"},
		{Kind: "class", Name: "Ignored"},
	}}
	graph := BuildCallGraph(module)
	if got := graph.FanOut()["a"]; got != 1 {
		t.Fatalf("FanOut(a) = %d, want 1", got)
	}
	if got := graph.FanIn()["b"]; got != 1 {
		t.Fatalf("FanIn(b) = %d, want distinct-caller count 1", got)
	}
	if got := graph.Outgoing("a"); len(got) != 2 {
		t.Fatalf("Outgoing(a) has %d edges, want 2 source occurrences", len(got))
	}
	if got := graph.Incoming("b"); len(got) != 2 {
		t.Fatalf("Incoming(b) has %d edges, want 2 source occurrences", len(got))
	}
	if got, err := graph.HighFanInNodes(1); err != nil || !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("HighFanInNodes(1) = %#v, %v", got, err)
	}
	if _, err := graph.HighFanInNodes(0); err == nil {
		t.Fatal("zero fan-in threshold must be rejected")
	}
	if got := graph.Cycles(); !reflect.DeepEqual(got, [][]string{{"a", "b", "c", "a"}}) {
		t.Fatalf("Cycles() = %#v", got)
	}
	depth := 2
	bounded, err := graph.ReachableFrom("a", &depth)
	if err != nil {
		t.Fatal(err)
	}
	want := []commonir.Relation{
		{Caller: "a", Callee: "b", CallType: "direct"},
		{Caller: "a", Callee: "b", CallType: "direct"},
		{Caller: "b", Callee: "c", CallType: "direct"},
	}
	if !reflect.DeepEqual(bounded.Edges(), want) {
		t.Fatalf("bounded edges = %#v, want %#v", bounded.Edges(), want)
	}
}

func TestCallGraphKeepsIsolatedNodesAndRejectsInvalidDepth(t *testing.T) {
	graph, err := NewCallGraph([]string{"alone"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(graph.Nodes(), []string{"alone"}) {
		t.Fatalf("Nodes() = %#v", graph.Nodes())
	}
	if graph.FanIn()["alone"] != 0 || graph.FanOut()["alone"] != 0 {
		t.Fatal("isolated-node degree must be zero")
	}
	depth := -1
	if _, err := graph.ReachableFrom("alone", &depth); err == nil {
		t.Fatal("negative depth must be rejected")
	}
}

func TestQualifiedCallersUseCommonIRParent(t *testing.T) {
	parent := "Service"
	graph := BuildCallGraph(commonir.Module{Entities: []commonir.Entity{{
		Kind: "method", Name: "run", Parent: &parent, Calls: []string{"helper"},
	}}})
	want := []commonir.Relation{{Caller: "Service.run", Callee: "helper", CallType: "direct"}}
	if !reflect.DeepEqual(graph.Edges(), want) {
		t.Fatalf("Edges() = %#v, want %#v", graph.Edges(), want)
	}
}

