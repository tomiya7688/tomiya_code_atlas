package generators

import (
	"fmt"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
)

//	{
//	  責務: [DeploymentNode: topology nodeまたはseries境界stubを保持する]
//	  フィールド: [ID: node識別子, Label: 表示名, Kind: node種別, Confidence: 解析確度, Boundary: 別diagramのstub, ReferenceOutput: 続きのdiagram名]
//	}
type DeploymentNode struct {
	ID              string            `json:"id"`
	Label           string            `json:"label"`
	Kind            string            `json:"kind"`
	Confidence      string            `json:"confidence"`
	Environment     string            `json:"environment,omitempty"`
	Source          string            `json:"source,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Boundary        bool              `json:"boundary,omitempty"`
	ReferenceOutput string            `json:"reference_output,omitempty"`
}

type DeploymentConnection struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Relation   string `json:"relation"`
	Confidence string `json:"confidence"`
}

//	{
//	  責務: [DeploymentDiagram: deployment topologyとseries境界参照を保持する]
//	  フィールド: [Name: 図名, Nodes: workloadやstub, Connections: deployment relation, References: 続きのdiagram参照]
//	}
type DeploymentDiagram struct {
	Name        string                 `json:"name"`
	Nodes       []DeploymentNode       `json:"nodes"`
	Connections []DeploymentConnection `json:"connections"`
	References  []DiagramReference     `json:"references,omitempty"`
}

//	{
//	  責務: [DeploymentDiagramBundle: 分割済みdeployment図と配置情報を保持する]
//	  フィールド: [Diagrams: 論理diagram一覧, Statistics: 分割指標, Placements: 系列階層と配置先]
//	}
type DeploymentDiagramBundle struct {
	Diagrams   []DeploymentDiagram `json:"diagrams"`
	Statistics map[string]any      `json:"statistics"`
	Placements []OutputPlacement   `json:"placements,omitempty"`
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
	partition, err := analyzers.PartitionGraph(graph, fanInThreshold, analyzers.DefaultMinChildSeriesSize)
	if err != nil {
		return DeploymentDiagramBundle{}, err
	}
	sharedOutputs := map[string]string{}
	for _, shared := range partition.Shared {
		sharedOutputs[shared] = stableOutputName("shared", shared, "deployment")
	}
	seriesOutputs := map[string]string{}
	for index, series := range partition.Series {
		name := stableOutputName(fmt.Sprintf("series_%d", index+1), partition.SeriesRoots[index], "deployment")
		for _, id := range series {
			seriesOutputs[id] = name
		}
	}
	makeDiagram := func(prefix, logicalName string, selected map[string]struct{}, includeOutgoing bool) DeploymentDiagram {
		d := DeploymentDiagram{Name: stableOutputName(prefix, logicalName, "deployment")}
		ids := map[string]struct{}{}
		for id := range selected {
			ids[id] = struct{}{}
		}
		for _, connection := range topology.Connections {
			_, sourceSelected := selected[connection.Source]
			_, targetSelected := selected[connection.Target]
			if includeOutgoing && sourceSelected && !targetSelected {
				ids[connection.Target] = struct{}{}
			}
		}
		for _, id := range sortedSet(ids) {
			if node, ok := nodeByID[id]; ok {
				model := deploymentNodeModel(node)
				_, isSelected := selected[id]
				model.Boundary = !isSelected
				if !isSelected {
					model.ReferenceOutput = seriesOutputs[id]
					if model.ReferenceOutput == "" {
						model.ReferenceOutput = sharedOutputs[id]
					}
				}
				d.Nodes = append(d.Nodes, model)
			}
		}
		for _, connection := range topology.Connections {
			_, a := selected[connection.Source]
			_, b := selected[connection.Target]
			_, targetIncluded := ids[connection.Target]
			if a && targetIncluded && (b || includeOutgoing) {
				d.Connections = append(d.Connections, deploymentConnectionModel(connection))
			}
		}
		return d
	}
	result := DeploymentDiagramBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		if len(series) > 0 {
			diagram := makeDiagram(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], stringSet(series), true)
			diagram.References = partitionReferences(partition, i, sharedOutputs, seriesOutputs)
			result.Diagrams = append(result.Diagrams, diagram)
			result.Placements = append(result.Placements, regularPlacement(diagram.Name, partition, i))
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
		diagram := makeDiagram("shared", shared, ids, false)
		for index := range diagram.Nodes {
			if diagram.Nodes[index].ID == shared {
				continue
			}
			diagram.Nodes[index].Boundary = true
			diagram.Nodes[index].ReferenceOutput = seriesOutputs[diagram.Nodes[index].ID]
			if diagram.Nodes[index].ReferenceOutput == "" {
				diagram.Nodes[index].ReferenceOutput = sharedOutputs[diagram.Nodes[index].ID]
			}
		}
		diagram.References = sharedDiagramReferences(shared, partition, sharedOutputs, seriesOutputs)
		result.Diagrams = append(result.Diagrams, diagram)
		result.Placements = append(result.Placements, sharedPlacement(diagram.Name, shared))
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
