package analyzers

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const deploymentTopologyFixturePath = "../../../tests/fixtures/analyzers/deployment_topology_v1.json"

func TestDeploymentTopologyMatchesSharedGolden(t *testing.T) {
	type componentModule struct {
		Name      string   `json:"name"`
		Component string   `json:"component"`
		Imports   []string `json:"imports"`
	}
	var fixture struct {
		ComponentModules []componentModule `json:"component_modules"`
		Merge            struct {
			Topologies []DeploymentTopology `json:"topologies"`
			Expected   DeploymentTopology   `json:"expected"`
		} `json:"merge"`
		ExpectedComponentTopology DeploymentTopology `json:"expected_component_topology"`
	}
	data, err := os.ReadFile(deploymentTopologyFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	merged := MergeDeploymentTopologies(fixture.Merge.Topologies...)
	if !reflect.DeepEqual(merged, fixture.Merge.Expected) {
		t.Fatalf("merged topology = %#v", merged)
	}

	units := make([]ComponentDependencyUnit, 0, len(fixture.ComponentModules))
	for _, item := range fixture.ComponentModules {
		units = append(units, ComponentDependencyUnit{Module: ModuleDependencyUnit{Name: item.Name, CommonIR: commonir.Module{Language: "python", Entities: []commonir.Entity{}, Imports: item.Imports, Diagnostics: []commonir.Diagnostic{}}}, Component: item.Component})
	}
	got := DeploymentTopologyFromComponentGraph(BuildComponentDependencyGraph(units))
	if !reflect.DeepEqual(got, fixture.ExpectedComponentTopology) {
		t.Fatalf("component topology = %#v", got)
	}
}

func TestMergeDeploymentTopologiesDoesNotMutateInputMetadata(t *testing.T) {
	metadata := map[string]string{"image": "api:1"}
	node := DeploymentNode{ID: "api", Confidence: ConfidenceConfirmed, Metadata: metadata}
	merged := MergeDeploymentTopologies(DeploymentTopology{Nodes: []DeploymentNode{node}})
	merged.Nodes[0].Metadata["image"] = "mutated"
	if metadata["image"] != "api:1" {
		t.Fatal("merge result aliases input metadata")
	}
}

