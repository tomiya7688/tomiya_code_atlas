package generators

import (
	"fmt"
	"sort"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

type ObjectField struct{ Name, Value string }

//	{
//	  責務: [ObjectNode: object instanceまたは系列境界の参照stubを保持する]
//	  フィールド: [Key: stable object key, Name: object名, TypeName: 型名, Fields: 表示field, External: 別diagramのnodeであること, ReferenceOutput: 続きのdiagram名]
//	}
type ObjectNode struct {
	Key, Name, TypeName string
	Fields              []ObjectField
	External            bool
	ReferenceOutput     string
}
type ObjectReference struct{ Source, Target, Label string }
type ObjectDiagram struct {
	Name       string
	Nodes      []ObjectNode
	References []ObjectReference
}
type ObjectDiagramBundle struct {
	Diagrams   []ObjectDiagram
	Statistics map[string]any
	Placements []OutputPlacement
}

//	{
//	  責務: [BuildObjectDiagramBundle: Common IR object factsと解決済み参照をpartitioned logical modelへ変換する]
//	}
func BuildObjectDiagramBundle(module commonir.Module, fanInThreshold int) (ObjectDiagramBundle, error) {
	graph := analyzers.BuildObjectRelationGraph(module)
	pairs := make([][2]string, 0, len(graph.Edges()))
	for _, edge := range graph.Edges() {
		pairs = append(pairs, [2]string{edge.Caller, edge.Callee})
	}
	relation, err := relationGraph(graph.Nodes(), pairs)
	if err != nil {
		return ObjectDiagramBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, analyzers.DefaultMinChildSeriesSize)
	if err != nil {
		return ObjectDiagramBundle{}, err
	}
	instances := map[string]commonir.ObjectInstance{}
	for _, instance := range module.Objects {
		instances[objectKey(instance)] = instance
	}
	nodeModel := func(key string) ObjectNode {
		instance, exists := instances[key]
		if !exists {
			return ObjectNode{Key: key, Name: key, TypeName: "external", External: true}
		}
		node := ObjectNode{Key: key, Name: instance.Name, TypeName: instance.TypeName}
		for _, pair := range instance.Values {
			if len(pair) >= 2 {
				node.Fields = append(node.Fields, ObjectField{Name: pair[0], Value: pair[1]})
			}
		}
		return node
	}
	sharedOutputs := map[string]string{}
	for _, shared := range partition.Shared {
		sharedOutputs[shared] = stableOutputName("shared", shared, "object")
	}
	seriesOutputs := map[string]string{}
	for index, series := range partition.Series {
		name := stableOutputName(fmt.Sprintf("series_%d", index+1), partition.SeriesRoots[index], "object")
		for _, key := range series {
			seriesOutputs[key] = name
		}
	}
	result := ObjectDiagramBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		selected := stringSet(series)
		references := []ObjectReference{}
		boundaryNodes := map[string]ObjectNode{}
		for _, edge := range graph.Edges() {
			_, sourceInside := selected[edge.Caller]
			if sourceInside {
				references = append(references, ObjectReference{edge.Caller, edge.Callee, edge.Label})
				if _, targetInside := selected[edge.Callee]; !targetInside {
					referenceOutput := seriesOutputs[edge.Callee]
					if referenceOutput == "" {
						referenceOutput = sharedOutputs[edge.Callee]
					}
					boundaryNode := nodeModel(edge.Callee)
					boundaryNode.Fields = []ObjectField{}
					boundaryNode.External = true
					boundaryNode.ReferenceOutput = referenceOutput
					boundaryNodes[edge.Callee] = boundaryNode
				}
			}
		}
		nodes := make([]ObjectNode, 0, len(series))
		for _, key := range series {
			nodes = append(nodes, nodeModel(key))
		}
		boundaryKeys := make([]string, 0, len(boundaryNodes))
		for key := range boundaryNodes {
			boundaryKeys = append(boundaryKeys, key)
		}
		sort.Strings(boundaryKeys)
		for _, key := range boundaryKeys {
			nodes = append(nodes, boundaryNodes[key])
		}
		name := seriesOutputs[partition.SeriesRoots[i]]
		result.Diagrams = append(result.Diagrams, ObjectDiagram{Name: name, Nodes: nodes, References: references})
		result.Placements = append(result.Placements, regularPlacement(name, partition, i))
	}
	for _, shared := range partition.Shared {
		selected := map[string]ObjectNode{shared: nodeModel(shared)}
		references := []ObjectReference{}
		for _, edge := range graph.Edges() {
			if edge.Caller == shared || edge.Callee == shared {
				for _, endpoint := range []string{edge.Caller, edge.Callee} {
					if endpoint == shared {
						continue
					}
					boundaryNode := nodeModel(endpoint)
					boundaryNode.Fields = []ObjectField{}
					boundaryNode.External = true
					boundaryNode.ReferenceOutput = seriesOutputs[endpoint]
					if boundaryNode.ReferenceOutput == "" {
						boundaryNode.ReferenceOutput = sharedOutputs[endpoint]
					}
					selected[endpoint] = boundaryNode
				}
				references = append(references, ObjectReference{edge.Caller, edge.Callee, edge.Label})
			}
		}
		nodes := []ObjectNode{}
		selectedKeys := make([]string, 0, len(selected))
		for key := range selected {
			selectedKeys = append(selectedKeys, key)
		}
		sort.Strings(selectedKeys)
		for _, key := range selectedKeys {
			nodes = append(nodes, selected[key])
		}
		name := sharedOutputs[shared]
		result.Diagrams = append(result.Diagrams, ObjectDiagram{Name: name, Nodes: nodes, References: references})
		result.Placements = append(result.Placements, sharedPlacement(name, shared))
	}
	return result, nil
}
