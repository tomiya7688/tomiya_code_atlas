package generators

import (
	"fmt"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
)

type PackageDiagramEdge struct{ Source, Target string }
type PackageDiagram struct {
	Name          string
	Nodes         []string
	Edges         []PackageDiagramEdge
	CycleNodes    []string
	IsolatedNodes []string
}
type PackageDiagramBundle struct {
	Diagrams   []PackageDiagram
	Statistics map[string]any
	Placements []OutputPlacement
}

func BuildPackageDiagramBundle(graph analyzers.PackageDependencyGraph, fanInThreshold int) (PackageDiagramBundle, error) {
	edges := graph.Edges()
	pairs := make([][2]string, 0, len(edges))
	for _, edge := range edges {
		pairs = append(pairs, [2]string{edge.Caller, edge.Callee})
	}
	relation, err := relationGraph(graph.Nodes(), pairs)
	if err != nil {
		return PackageDiagramBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, 3)
	if err != nil {
		return PackageDiagramBundle{}, err
	}
	cycles, err := graph.Cycles()
	if err != nil {
		return PackageDiagramBundle{}, err
	}
	cycleNodes := map[string]struct{}{}
	for _, cycle := range cycles {
		for _, node := range cycle {
			cycleNodes[node] = struct{}{}
		}
	}
	isolated := stringSet(graph.IsolatedNodes())
	makeDiagram := func(prefix, logicalName string, names map[string]struct{}) PackageDiagram {
		d := PackageDiagram{Name: stableOutputName(prefix, logicalName, "package"), Nodes: sortedSet(names)}
		for _, edge := range edges {
			_, a := names[edge.Caller]
			_, b := names[edge.Callee]
			if a && b {
				d.Edges = append(d.Edges, PackageDiagramEdge{edge.Caller, edge.Callee})
			}
		}
		for _, node := range d.Nodes {
			if _, ok := cycleNodes[node]; ok {
				d.CycleNodes = append(d.CycleNodes, node)
			}
			if _, ok := isolated[node]; ok {
				d.IsolatedNodes = append(d.IsolatedNodes, node)
			}
		}
		return d
	}
	result := PackageDiagramBundle{Statistics: partitionStatistics(partition)}
	result.Statistics["isolated_node_count"] = len(isolated)
	for i, series := range partition.Series {
		if len(series) == 0 {
			continue
		}
		d := makeDiagram(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], stringSet(series))
		result.Diagrams = append(result.Diagrams, d)
		result.Placements = append(result.Placements, regularPlacement(d.Name, partition, i))
	}
	for _, shared := range partition.Shared {
		names := map[string]struct{}{shared: {}}
		for _, edge := range edges {
			if edge.Callee == shared || edge.Caller == shared {
				names[edge.Caller] = struct{}{}
				names[edge.Callee] = struct{}{}
			}
		}
		d := makeDiagram("shared", shared, names)
		result.Diagrams = append(result.Diagrams, d)
		result.Placements = append(result.Placements, sharedPlacement(d.Name, shared))
	}
	return result, nil
}

type ComponentDiagramNode struct {
	Name     string
	External bool
}
type ComponentDiagramEdge struct{ Source, Target string }
type ComponentDiagram struct {
	Name       string
	Nodes      []ComponentDiagramNode
	Edges      []ComponentDiagramEdge
	CycleNodes []string
}
type ComponentDiagramBundle struct {
	Diagrams   []ComponentDiagram
	Statistics map[string]any
	Placements []OutputPlacement
}

func BuildComponentDiagramBundle(graph analyzers.ComponentDependencyGraph, fanInThreshold int) (ComponentDiagramBundle, error) {
	edges := graph.Edges()
	pairs := make([][2]string, 0, len(edges))
	for _, edge := range edges {
		pairs = append(pairs, [2]string{edge.Caller, edge.Callee})
	}
	relation, err := relationGraph(graph.Nodes(), pairs)
	if err != nil {
		return ComponentDiagramBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, 3)
	if err != nil {
		return ComponentDiagramBundle{}, err
	}
	cycleNodes := map[string]struct{}{}
	for _, cycle := range graph.Cycles() {
		for _, node := range cycle {
			cycleNodes[node] = struct{}{}
		}
	}
	external := stringSet(graph.ExternalNodes())
	makeDiagram := func(prefix, logicalName string, names map[string]struct{}) ComponentDiagram {
		d := ComponentDiagram{Name: stableOutputName(prefix, logicalName, "component")}
		for _, name := range sortedSet(names) {
			_, isExternal := external[name]
			d.Nodes = append(d.Nodes, ComponentDiagramNode{Name: name, External: isExternal})
			if _, ok := cycleNodes[name]; ok {
				d.CycleNodes = append(d.CycleNodes, name)
			}
		}
		for _, edge := range edges {
			_, a := names[edge.Caller]
			_, b := names[edge.Callee]
			if a && b {
				d.Edges = append(d.Edges, ComponentDiagramEdge{edge.Caller, edge.Callee})
			}
		}
		return d
	}
	result := ComponentDiagramBundle{Statistics: partitionStatistics(partition)}
	result.Statistics["external_node_count"] = len(external)
	for i, series := range partition.Series {
		if len(series) == 0 {
			continue
		}
		d := makeDiagram(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], stringSet(series))
		result.Diagrams = append(result.Diagrams, d)
		result.Placements = append(result.Placements, regularPlacement(d.Name, partition, i))
	}
	for _, shared := range partition.Shared {
		names := map[string]struct{}{shared: {}}
		for _, edge := range edges {
			if edge.Callee == shared || edge.Caller == shared {
				names[edge.Caller] = struct{}{}
				names[edge.Callee] = struct{}{}
			}
		}
		d := makeDiagram("shared", shared, names)
		result.Diagrams = append(result.Diagrams, d)
		result.Placements = append(result.Placements, sharedPlacement(d.Name, shared))
	}
	return result, nil
}
