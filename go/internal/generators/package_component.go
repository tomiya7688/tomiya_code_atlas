package generators

import (
	"fmt"
	"sort"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
)

type PackageDiagramEdge struct{ Source, Target string }

//	{
//	  責務: [PackageDiagram: package dependencyとseries境界参照を保持する]
//	  フィールド: [Name: diagram名, Nodes: 表示node, Edges: dependency relation, ExternalNodes: 境界stub, References: 続きのdiagram参照]
//	}
type PackageDiagram struct {
	Name          string
	Nodes         []string
	Edges         []PackageDiagramEdge
	ExternalNodes []string
	References    []DiagramReference
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
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, analyzers.DefaultMinChildSeriesSize)
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
	sharedOutputs := map[string]string{}
	for _, shared := range partition.Shared {
		sharedOutputs[shared] = stableOutputName("shared", shared, "package")
	}
	seriesOutputs := map[string]string{}
	for index, series := range partition.Series {
		name := stableOutputName(fmt.Sprintf("series_%d", index+1), partition.SeriesRoots[index], "package")
		for _, node := range series {
			seriesOutputs[node] = name
		}
	}
	makeDiagram := func(prefix, logicalName string, names map[string]struct{}, includeOutgoing bool) PackageDiagram {
		d := PackageDiagram{Name: stableOutputName(prefix, logicalName, "package"), Nodes: sortedSet(names)}
		externalNodes := map[string]struct{}{}
		for _, edge := range edges {
			_, a := names[edge.Caller]
			_, b := names[edge.Callee]
			if a && b {
				d.Edges = append(d.Edges, PackageDiagramEdge{edge.Caller, edge.Callee})
			} else if includeOutgoing && a {
				d.Edges = append(d.Edges, PackageDiagramEdge{edge.Caller, edge.Callee})
				externalNodes[edge.Callee] = struct{}{}
			}
		}
		d.ExternalNodes = sortedSet(externalNodes)
		for _, node := range d.ExternalNodes {
			d.Nodes = append(d.Nodes, node)
		}
		sort.Strings(d.Nodes)
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
		d := makeDiagram(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], stringSet(series), true)
		d.References = partitionReferences(partition, i, sharedOutputs, seriesOutputs)
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
		d := makeDiagram("shared", shared, names, false)
		for _, node := range d.Nodes {
			if node != shared {
				d.ExternalNodes = append(d.ExternalNodes, node)
			}
		}
		d.References = sharedDiagramReferences(shared, partition, sharedOutputs, seriesOutputs)
		result.Diagrams = append(result.Diagrams, d)
		result.Placements = append(result.Placements, sharedPlacement(d.Name, shared))
	}
	return result, nil
}

//	{
//	  責務: [ComponentDiagramNode: componentまたは外部参照stubを保持する]
//	  フィールド: [Name: component名, External: 解析対象外のcomponent, Boundary: 別seriesのstub, ReferenceOutput: 続きのdiagram名]
//	}
type ComponentDiagramNode struct {
	Name            string
	External        bool
	Boundary        bool
	ReferenceOutput string
}
type ComponentDiagramEdge struct{ Source, Target string }

//	{
//	  責務: [ComponentDiagram: component relation、cycle、series境界参照を保持する]
//	  フィールド: [Name: 図名, Nodes: component node, Edges: 依存relation, References: 続きのdiagram参照]
//	}
type ComponentDiagram struct {
	Name       string
	Nodes      []ComponentDiagramNode
	Edges      []ComponentDiagramEdge
	References []DiagramReference
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
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, analyzers.DefaultMinChildSeriesSize)
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
	sharedOutputs := map[string]string{}
	for _, shared := range partition.Shared {
		sharedOutputs[shared] = stableOutputName("shared", shared, "component")
	}
	seriesOutputs := map[string]string{}
	for index, series := range partition.Series {
		name := stableOutputName(fmt.Sprintf("series_%d", index+1), partition.SeriesRoots[index], "component")
		for _, node := range series {
			seriesOutputs[node] = name
		}
	}
	makeDiagram := func(prefix, logicalName string, names map[string]struct{}, includeOutgoing bool) ComponentDiagram {
		d := ComponentDiagram{Name: stableOutputName(prefix, logicalName, "component")}
		primary := map[string]struct{}{}
		for name := range names {
			primary[name] = struct{}{}
		}
		for _, edge := range edges {
			_, sourceSelected := primary[edge.Caller]
			_, targetSelected := primary[edge.Callee]
			if sourceSelected && targetSelected {
				d.Edges = append(d.Edges, ComponentDiagramEdge{edge.Caller, edge.Callee})
			} else if includeOutgoing && sourceSelected {
				d.Edges = append(d.Edges, ComponentDiagramEdge{edge.Caller, edge.Callee})
				names[edge.Callee] = struct{}{}
			}
		}
		for _, name := range sortedSet(names) {
			_, isExternal := external[name]
			_, isBoundary := primary[name]
			_, isShared := sharedOutputs[name]
			referenceOutput := seriesOutputs[name]
			if isShared {
				referenceOutput = sharedOutputs[name]
			}
			d.Nodes = append(d.Nodes, ComponentDiagramNode{Name: name, External: isExternal, Boundary: !isBoundary, ReferenceOutput: referenceOutput})
			if _, ok := cycleNodes[name]; ok {
				d.CycleNodes = append(d.CycleNodes, name)
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
		d := makeDiagram(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], stringSet(series), true)
		d.References = partitionReferences(partition, i, sharedOutputs, seriesOutputs)
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
		d := makeDiagram("shared", shared, names, false)
		for index := range d.Nodes {
			if d.Nodes[index].Name == shared {
				continue
			}
			d.Nodes[index].Boundary = true
			d.Nodes[index].ReferenceOutput = seriesOutputs[d.Nodes[index].Name]
			if d.Nodes[index].ReferenceOutput == "" {
				d.Nodes[index].ReferenceOutput = sharedOutputs[d.Nodes[index].Name]
			}
		}
		d.References = sharedDiagramReferences(shared, partition, sharedOutputs, seriesOutputs)
		result.Diagrams = append(result.Diagrams, d)
		result.Placements = append(result.Placements, sharedPlacement(d.Name, shared))
	}
	return result, nil
}
