package generators

import (
	"fmt"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

type ObjectField struct{ Name, Value string }
type ObjectNode struct {
	Key, Name, TypeName string
	Fields              []ObjectField
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
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, 3)
	if err != nil {
		return ObjectDiagramBundle{}, err
	}
	instances := map[string]commonir.ObjectInstance{}
	for _, instance := range module.Objects {
		instances[objectKey(instance)] = instance
	}
	nodeModel := func(key string) ObjectNode {
		instance := instances[key]
		node := ObjectNode{Key: key, Name: instance.Name, TypeName: instance.TypeName}
		for _, pair := range instance.Values {
			if len(pair) >= 2 {
				node.Fields = append(node.Fields, ObjectField{Name: pair[0], Value: pair[1]})
			}
		}
		return node
	}
	result := ObjectDiagramBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		selected := stringSet(series)
		references := []ObjectReference{}
		for _, edge := range graph.Edges() {
			_, a := selected[edge.Caller]
			_, b := selected[edge.Callee]
			if a && b {
				references = append(references, ObjectReference{edge.Caller, edge.Callee, edge.Label})
			}
		}
		nodes := make([]ObjectNode, 0, len(series))
		for _, key := range series {
			nodes = append(nodes, nodeModel(key))
		}
		name := stableOutputName(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], "object")
		result.Diagrams = append(result.Diagrams, ObjectDiagram{Name: name, Nodes: nodes, References: references})
		result.Placements = append(result.Placements, regularPlacement(name, partition, i))
	}
	for _, shared := range partition.Shared {
		selected := map[string]struct{}{shared: {}}
		references := []ObjectReference{}
		for _, edge := range graph.Edges() {
			if edge.Caller == shared || edge.Callee == shared {
				selected[edge.Caller] = struct{}{}
				selected[edge.Callee] = struct{}{}
				references = append(references, ObjectReference{edge.Caller, edge.Callee, edge.Label})
			}
		}
		nodes := []ObjectNode{}
		for _, key := range sortedSet(selected) {
			nodes = append(nodes, nodeModel(key))
		}
		name := stableOutputName("shared", shared, "object")
		result.Diagrams = append(result.Diagrams, ObjectDiagram{Name: name, Nodes: nodes, References: references})
		result.Placements = append(result.Placements, sharedPlacement(name, shared))
	}
	return result, nil
}
