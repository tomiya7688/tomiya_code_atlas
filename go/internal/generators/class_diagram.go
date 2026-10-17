package generators

import (
	"fmt"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

type ClassMember struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters"`
	Visibility string   `json:"visibility"`
}

type ClassNode struct {
	Name     string        `json:"name"`
	Members  []ClassMember `json:"members"`
	External bool          `json:"external,omitempty"`
}

type ClassRelation struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type ClassDiagram struct {
	Name      string          `json:"name"`
	Nodes     []ClassNode     `json:"nodes"`
	Relations []ClassRelation `json:"relations"`
}

type ClassDiagramBundle struct {
	Diagrams   []ClassDiagram    `json:"diagrams"`
	Statistics map[string]any    `json:"statistics"`
	Placements []OutputPlacement `json:"placements,omitempty"`
}

type ClassDiagramOptions struct {
	IncludePublic      bool
	IncludeProtected   bool
	IncludeInternal    bool
	IncludePrivate     bool
	IncludeMethods     bool
	IncludeInheritance bool
	IncludeUses        bool
}

func DefaultClassDiagramOptions() ClassDiagramOptions {
	return ClassDiagramOptions{true, true, true, true, true, true, true}
}

func (options ClassDiagramOptions) includesVisibility(visibility string) bool {
	switch visibility {
	case "public":
		return options.IncludePublic
	case "protected":
		return options.IncludeProtected
	case "internal":
		return options.IncludeInternal
	case "private":
		return options.IncludePrivate
	default:
		return true
	}
}

//	{
//	  責務: [BuildClassDiagramBundle: class relationとmember情報からrenderer-neutralなclass図bundleを作る]
//	}
func BuildClassDiagramBundle(module commonir.Module, fanInThreshold int, options *ClassDiagramOptions) (ClassDiagramBundle, error) {
	resolvedOptions := DefaultClassDiagramOptions()
	if options != nil {
		resolvedOptions = *options
	}
	graph := analyzers.BuildClassRelationGraph(module)
	edges := graph.Edges()
	pairs := make([][2]string, 0, len(edges))
	for _, edge := range edges {
		pairs = append(pairs, [2]string{edge.Caller, edge.Callee})
	}
	relation, err := relationGraph(graph.Nodes(), pairs)
	if err != nil {
		return ClassDiagramBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, 3)
	if err != nil {
		return ClassDiagramBundle{}, err
	}
	classes := map[string]commonir.Entity{}
	methods := map[string][]commonir.Entity{}
	for _, entity := range module.Entities {
		name := qualifiedEntityName(entity)
		if entity.Kind == "class" {
			classes[name] = entity
		}
		if entity.Kind == "method" && entity.Parent != nil {
			methods[*entity.Parent] = append(methods[*entity.Parent], entity)
		}
	}
	visible := make([]analyzers.ClassRelationEdge, 0, len(edges))
	for _, edge := range edges {
		if edge.Relation == "inheritance" && !resolvedOptions.IncludeInheritance {
			continue
		}
		if edge.Relation == "uses" && !resolvedOptions.IncludeUses {
			continue
		}
		visible = append(visible, edge)
	}
	nodeModel := func(name string) ClassNode {
		entity, ok := classes[name]
		if !ok {
			return ClassNode{Name: name, Members: []ClassMember{}, External: true}
		}
		node := ClassNode{Name: name, Members: []ClassMember{}}
		if resolvedOptions.IncludeMethods {
			for _, method := range methods[name] {
				if !resolvedOptions.includesVisibility(method.Visibility) {
					continue
				}
				visibility := method.Visibility
				if visibility == "" {
					visibility = "unspecified"
				}
				node.Members = append(node.Members, ClassMember{Name: method.Name, Parameters: append([]string(nil), method.Parameters...), Visibility: visibility})
			}
		}
		_ = entity
		return node
	}
	relationModel := func(edge analyzers.ClassRelationEdge) ClassRelation {
		return ClassRelation{Source: edge.Caller, Target: edge.Callee, Kind: edge.Relation}
	}
	result := ClassDiagramBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		selected := stringSet(series)
		relations := []ClassRelation{}
		for _, edge := range visible {
			_, sourceOK := selected[edge.Caller]
			_, targetOK := selected[edge.Callee]
			if sourceOK && targetOK {
				relations = append(relations, relationModel(edge))
			}
		}
		nodes := make([]ClassNode, 0, len(series))
		for _, name := range series {
			nodes = append(nodes, nodeModel(name))
		}
		name := stableOutputName(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], "class")
		result.Diagrams = append(result.Diagrams, ClassDiagram{Name: name, Nodes: nodes, Relations: relations})
		result.Placements = append(result.Placements, regularPlacement(name, partition, i))
	}
	for _, shared := range partition.Shared {
		incoming := []analyzers.ClassRelationEdge{}
		selected := map[string]struct{}{shared: {}}
		for _, edge := range visible {
			if edge.Callee == shared {
				incoming = append(incoming, edge)
				selected[edge.Caller] = struct{}{}
			}
		}
		nodes := []ClassNode{}
		for _, name := range sortedSet(selected) {
			nodes = append(nodes, nodeModel(name))
		}
		relations := make([]ClassRelation, 0, len(incoming))
		for _, edge := range incoming {
			relations = append(relations, relationModel(edge))
		}
		name := stableOutputName("shared", shared, "class")
		result.Diagrams = append(result.Diagrams, ClassDiagram{Name: name, Nodes: nodes, Relations: relations})
		result.Placements = append(result.Placements, sharedPlacement(name, shared))
	}
	return result, nil
}
