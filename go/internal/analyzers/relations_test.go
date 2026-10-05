package analyzers

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const relationsFixturePath = "../../../tests/fixtures/analyzers/relations_v1.json"

func TestClassObjectAndComponentRelationsMatchSharedGolden(t *testing.T) {
	type componentModule struct {
		Name      string   `json:"name"`
		Component string   `json:"component"`
		Imports   []string `json:"imports"`
	}
	type expected struct {
		ClassNodes             []string                  `json:"class_nodes"`
		ClassEdges             []ClassRelationEdge       `json:"class_edges"`
		ObjectNodes            []string                  `json:"object_nodes"`
		ObjectEdges            []ObjectRelationEdge      `json:"object_edges"`
		ObjectCycles           [][]string                `json:"object_cycles"`
		ComponentNodes         []string                  `json:"component_nodes"`
		ComponentEdges         []ComponentDependencyEdge `json:"component_edges"`
		ComponentExternalNodes []string                  `json:"component_external_nodes"`
		ComponentCycles        [][]string                `json:"component_cycles"`
	}
	var fixture struct {
		CommonIR         commonir.Module   `json:"common_ir"`
		ComponentModules []componentModule `json:"component_modules"`
		Expected         expected          `json:"expected"`
	}
	data, err := os.ReadFile(relationsFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	classGraph := BuildClassRelationGraph(fixture.CommonIR)
	if !reflect.DeepEqual(classGraph.Nodes(), fixture.Expected.ClassNodes) {
		t.Fatalf("class nodes = %#v", classGraph.Nodes())
	}
	if !reflect.DeepEqual(classGraph.Edges(), fixture.Expected.ClassEdges) {
		t.Fatalf("class edges = %#v", classGraph.Edges())
	}

	objectGraph := BuildObjectRelationGraph(fixture.CommonIR)
	if !reflect.DeepEqual(objectGraph.Nodes(), fixture.Expected.ObjectNodes) {
		t.Fatalf("object nodes = %#v", objectGraph.Nodes())
	}
	if !reflect.DeepEqual(objectGraph.Edges(), fixture.Expected.ObjectEdges) {
		t.Fatalf("object edges = %#v", objectGraph.Edges())
	}
	if !reflect.DeepEqual(objectGraph.Cycles(), fixture.Expected.ObjectCycles) {
		t.Fatalf("object cycles = %#v", objectGraph.Cycles())
	}

	componentUnits := make([]ComponentDependencyUnit, 0, len(fixture.ComponentModules))
	for _, item := range fixture.ComponentModules {
		componentUnits = append(componentUnits, ComponentDependencyUnit{
			Module: ModuleDependencyUnit{
				Name:     item.Name,
				CommonIR: commonir.Module{Language: "python", Entities: []commonir.Entity{}, Imports: item.Imports, Diagnostics: []commonir.Diagnostic{}},
			},
			Component: item.Component,
		})
	}
	componentGraph := BuildComponentDependencyGraph(componentUnits)
	if !reflect.DeepEqual(componentGraph.Nodes(), fixture.Expected.ComponentNodes) {
		t.Fatalf("component nodes = %#v", componentGraph.Nodes())
	}
	if !reflect.DeepEqual(componentGraph.Edges(), fixture.Expected.ComponentEdges) {
		t.Fatalf("component edges = %#v", componentGraph.Edges())
	}
	if !reflect.DeepEqual(componentGraph.ExternalNodes(), fixture.Expected.ComponentExternalNodes) {
		t.Fatalf("external nodes = %#v", componentGraph.ExternalNodes())
	}
	if !reflect.DeepEqual(componentGraph.Cycles(), fixture.Expected.ComponentCycles) {
		t.Fatalf("component cycles = %#v", componentGraph.Cycles())
	}
}

