package generators

import (
	"fmt"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
)

type DeploymentNode struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Kind        string            `json:"kind"`
	Confidence  string            `json:"confidence"`
	Environment string            `json:"environment,omitempty"`
	Source      string            `json:"source,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type DeploymentConnection struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Relation   string `json:"relation"`
	Confidence string `json:"confidence"`
}

type DeploymentDiagram struct {
	Name        string                 `json:"name"`
	Nodes       []DeploymentNode       `json:"nodes"`
	Connections []DeploymentConnection `json:"connections"`
}

type DeploymentDiagramBundle struct {
	Diagrams   []DeploymentDiagram `json:"diagrams"`
	Statistics map[string]any      `json:"statistics"`
}

//	{
//	  責務: [BuildDeploymentDiagramBundle: deployment topologyを共通partition規則でlogical diagramsへ分割する]
//	}
func BuildDeploymentDiagramBundle(topology analyzers.DeploymentTopology, fanInThreshold int) (DeploymentDiagramBundle, error) {
	nodes := make([]string, 0, len(topology.Nodes))
	nodeByID := map[string]analyzers.DeploymentNode{}
	for _, node := range topology.Nodes {
		nodes = append(nodes, node.ID)
		nodeByID[node.ID] = node
	}
	pairs := make([][2]string, 0, len(topology.Connections))
	for _, connection := range topology.Connections {
		pairs = append(pairs, [2]string{connection.Source, connection.Target})
	}
	graph, err := relationGraph(nodes, pairs)
	if err != nil {
		return DeploymentDiagramBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(graph, fanInThreshold, 3)
	if err != nil {
		return DeploymentDiagramBundle{}, err
	}
	makeDiagram := func(prefix, logicalName string, ids map[string]struct{}) DeploymentDiagram {
		d := DeploymentDiagram{Name: stableOutputName(prefix, logicalName, "deployment")}
		for _, id := range sortedSet(ids) {
			if node, ok := nodeByID[id]; ok {
				d.Nodes = append(d.Nodes, deploymentNodeModel(node))
			}
		}
		for _, connection := range topology.Connections {
			_, a := ids[connection.Source]
			_, b := ids[connection.Target]
			if a && b {
				d.Connections = append(d.Connections, deploymentConnectionModel(connection))
			}
		}
		return d
	}
	result := DeploymentDiagramBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		if len(series) > 0 {
			result.Diagrams = append(result.Diagrams, makeDiagram(fmt.Sprintf("series_%d", i+1), series[0], stringSet(series)))
		}
	}
	for _, shared := range partition.Shared {
		ids := map[string]struct{}{shared: {}}
		for _, connection := range topology.Connections {
			if connection.Target == shared || connection.Source == shared {
				ids[connection.Source] = struct{}{}
				ids[connection.Target] = struct{}{}
			}
		}
		result.Diagrams = append(result.Diagrams, makeDiagram("shared", shared, ids))
	}
	for _, node := range topology.Nodes {
		switch node.Confidence {
		case analyzers.ConfidenceConfirmed:
			result.Statistics["confirmed_node_count"] = intStatistic(result.Statistics, "confirmed_node_count") + 1
		case analyzers.ConfidenceInferred:
			result.Statistics["inferred_node_count"] = intStatistic(result.Statistics, "inferred_node_count") + 1
		default:
			result.Statistics["unknown_node_count"] = intStatistic(result.Statistics, "unknown_node_count") + 1
		}
	}
	for _, key := range []string{"confirmed_node_count", "inferred_node_count", "unknown_node_count"} {
		if _, ok := result.Statistics[key]; !ok {
			result.Statistics[key] = 0
		}
	}
	return result, nil
}

func deploymentNodeModel(node analyzers.DeploymentNode) DeploymentNode {
	metadata := make(map[string]string, len(node.Metadata))
	for key, value := range node.Metadata {
		metadata[key] = value
	}
	return DeploymentNode{ID: node.ID, Label: node.Label, Kind: node.Kind, Confidence: string(node.Confidence), Environment: node.Environment, Source: node.Source, Metadata: metadata}
}

func deploymentConnectionModel(connection analyzers.DeploymentConnection) DeploymentConnection {
	return DeploymentConnection{Source: connection.Source, Target: connection.Target, Relation: connection.Relation, Confidence: string(connection.Confidence)}
}

func intStatistic(values map[string]any, key string) int {
	if value, ok := values[key].(int); ok {
		return value
	}
	return 0
}
