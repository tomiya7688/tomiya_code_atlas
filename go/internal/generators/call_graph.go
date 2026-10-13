package generators

import (
	"fmt"
	"sort"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [CallGraphDiagram: 一つのpartitioned call graphと安定名を保持する]
//	}
type CallGraphEdge struct {
	Caller string `json:"caller"`
	Callee string `json:"callee"`
}

type CallGraph struct {
	Nodes []string        `json:"nodes"`
	Edges []CallGraphEdge `json:"edges"`
}

type CallGraphDiagram struct {
	Name  string    `json:"name"`
	Graph CallGraph `json:"graph"`
}

type CallGraphBundle struct {
	Diagrams   []CallGraphDiagram `json:"diagrams"`
	Statistics map[string]any     `json:"statistics"`
	Placements []OutputPlacement  `json:"placements,omitempty"`
}

//	{
//	  責務: [BuildCallGraph: Common IRからrenderer-neutralなcall graphを作り、任意rootで絞る]
//	  引数: [module: Common IR, root: nilまたは到達探索root, maxDepth: nilまたは最大edge深さ]
//	  戻り値: [CallGraph: 生成graph, error: 不正depth等]
//	}
func BuildCallGraph(module commonir.Module, root *string, maxDepth *int) (CallGraph, error) {
	graph, err := analyzedCallGraph(module, root, maxDepth)
	if err != nil {
		return CallGraph{}, err
	}
	return callGraphModel(graph), nil
}

func analyzedCallGraph(module commonir.Module, root *string, maxDepth *int) (analyzers.CallGraph, error) {
	graph := analyzers.BuildCallGraph(module)
	if root == nil {
		if maxDepth != nil && *maxDepth < 0 {
			return analyzers.CallGraph{}, fmt.Errorf("max depth must be non-negative or nil")
		}
		return graph, nil
	}
	return graph.ReachableFrom(*root, maxDepth)
}

func callGraphModel(graph analyzers.CallGraph) CallGraph {
	edges := graph.Edges()
	model := CallGraph{Nodes: graph.Nodes(), Edges: make([]CallGraphEdge, 0, len(edges))}
	for _, edge := range edges {
		model.Edges = append(model.Edges, CallGraphEdge{Caller: edge.Caller, Callee: edge.Callee})
	}
	return model
}

//	{
//	  責務: [BuildCallGraphBundle: call graphを通常seriesとshared hubへ分割する]
//	  処理: [1: call graphを構築, 2: 共通partitionを適用, 3: series境界calleeを表示用nodeとして残す, 4: shared diagramを作る]
//	}
func BuildCallGraphBundle(module commonir.Module, fanInThreshold int, root *string, maxDepth *int) (CallGraphBundle, error) {
	graph, err := analyzedCallGraph(module, root, maxDepth)
	if err != nil {
		return CallGraphBundle{}, err
	}
	relation, err := analyzers.RelationGraphFromCallGraph(graph)
	if err != nil {
		return CallGraphBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, 3)
	if err != nil {
		return CallGraphBundle{}, err
	}
	seriesByNode := map[string]int{}
	for i, series := range partition.Series {
		for _, node := range series {
			seriesByNode[node] = i
		}
	}
	shared := stringSet(partition.Shared)
	result := CallGraphBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		selected := stringSet(series)
		edges := make([]commonir.Relation, 0)
		boundary := map[string]struct{}{}
		for _, edge := range graph.Edges() {
			if _, ok := selected[edge.Caller]; !ok {
				continue
			}
			if _, ok := shared[edge.Callee]; ok {
				continue
			}
			_, inside := selected[edge.Callee]
			calleeSeries, calleeInPartition := seriesByNode[edge.Callee]
			if !inside && calleeInPartition && calleeSeries == seriesByNode[edge.Caller] {
				continue
			}
			edges = append(edges, edge)
			if !inside {
				boundary[edge.Callee] = struct{}{}
			}
		}
		nodes := append([]string(nil), series...)
		for node := range boundary {
			nodes = append(nodes, node)
		}
		sort.Strings(nodes)
		subgraph, err := analyzers.NewCallGraph(nodes, edges)
		if err != nil {
			return CallGraphBundle{}, err
		}
		name := stableOutputName(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], "call_graph")
		result.Diagrams = append(result.Diagrams, CallGraphDiagram{Name: name, Graph: callGraphModel(subgraph)})
		result.Placements = append(result.Placements, regularPlacement(name, partition, i))
	}
	for _, sharedNode := range partition.Shared {
		edges := make([]commonir.Relation, 0)
		nodes := map[string]struct{}{sharedNode: {}}
		for _, edge := range graph.Edges() {
			if edge.Callee != sharedNode && edge.Caller != sharedNode {
				continue
			}
			edges = append(edges, edge)
			nodes[edge.Caller] = struct{}{}
			nodes[edge.Callee] = struct{}{}
		}
		subgraph, err := analyzers.NewCallGraph(sortedSet(nodes), edges)
		if err != nil {
			return CallGraphBundle{}, err
		}
		name := stableOutputName("shared", sharedNode, "call_graph")
		result.Diagrams = append(result.Diagrams, CallGraphDiagram{Name: name, Graph: callGraphModel(subgraph)})
		result.Placements = append(result.Placements, sharedPlacement(name, sharedNode))
	}
	return result, nil
}
