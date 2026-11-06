package generators

import (
	"fmt"
	"sort"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

type ClassMember struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters"`
	Visibility string   `json:"visibility"`
}

//	{
//	  責務: [ClassNode: class図のclassまたは境界stubを保持する]
//	  フィールド: [Name: class名, Members: 表示するmember, External: 別diagramのnodeであること, ReferenceOutput: 続きのdiagram名]
//	}
type ClassNode struct {
	Name            string        `json:"name"`
	Members         []ClassMember `json:"members"`
	External        bool          `json:"external,omitempty"`
	ReferenceOutput string        `json:"reference_output,omitempty"`
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
	pairs := make([][2]string, 0, len(visible))
	for _, edge := range visible {
		pairs = append(pairs, [2]string{edge.Caller, edge.Callee})
	}
	relation, err := relationGraph(graph.Nodes(), pairs)
	if err != nil {
		return ClassDiagramBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, analyzers.DefaultMinChildSeriesSize)
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
	sharedOutputs := map[string]string{}
	for _, shared := range partition.Shared {
		sharedOutputs[shared] = stableOutputName("shared", shared, "class")
	}
	seriesOutputs := map[string]string{}
	for index, series := range partition.Series {
		name := stableOutputName(fmt.Sprintf("series_%d", index+1), partition.SeriesRoots[index], "class")
		for _, className := range series {
			seriesOutputs[className] = name
		}
	}
	result := ClassDiagramBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		selected := stringSet(series)
		relations := []ClassRelation{}
		nodesByName := map[string]ClassNode{}
		for _, edge := range visible {
			_, sourceOK := selected[edge.Caller]
			if sourceOK {
				relations = append(relations, relationModel(edge))
				if _, targetOK := selected[edge.Callee]; !targetOK {
					referenceOutput := seriesOutputs[edge.Callee]
					if referenceOutput == "" {
						referenceOutput = sharedOutputs[edge.Callee]
					}
					nodesByName[edge.Callee] = ClassNode{Name: edge.Callee, Members: []ClassMember{}, External: true, ReferenceOutput: referenceOutput}
				}
			}
		}
		nodes := make([]ClassNode, 0, len(series))
		for _, name := range series {
			nodes = append(nodes, nodeModel(name))
		}
		stubNames := make([]string, 0, len(nodesByName))
		for name := range nodesByName {
			stubNames = append(stubNames, name)
		}
		sort.Strings(stubNames)
		for _, name := range stubNames {
			nodes = append(nodes, nodesByName[name])
		}
		name := seriesOutputs[partition.SeriesRoots[i]]
		result.Diagrams = append(result.Diagrams, ClassDiagram{Name: name, Nodes: nodes, Relations: relations})
		result.Placements = append(result.Placements, regularPlacement(name, partition, i))
	}
	for _, shared := range partition.Shared {
		sharedRelations := []analyzers.ClassRelationEdge{}
		selected := map[string]ClassNode{shared: nodeModel(shared)}
		for _, edge := range visible {
			if edge.Callee == shared || edge.Caller == shared {
				sharedRelations = append(sharedRelations, edge)
				otherNode := edge.Caller
				if otherNode == shared {
					otherNode = edge.Callee
				}
				if otherNode == shared {
					continue
				}
				if _, isShared := sharedOutputs[otherNode]; isShared {
					selected[otherNode] = ClassNode{Name: otherNode, Members: []ClassMember{}, External: true, ReferenceOutput: sharedOutputs[otherNode]}
				} else {
					selected[otherNode] = ClassNode{Name: otherNode, Members: []ClassMember{}, External: true, ReferenceOutput: seriesOutputs[otherNode]}
				}
			}
		}
		nodes := make([]ClassNode, 0, len(selected))
		selectedNames := make([]string, 0, len(selected))
		for name := range selected {
			selectedNames = append(selectedNames, name)
		}
		sort.Strings(selectedNames)
		for _, name := range selectedNames {
			nodes = append(nodes, selected[name])
		}
		relations := make([]ClassRelation, 0, len(sharedRelations))
		for _, edge := range sharedRelations {
			relations = append(relations, relationModel(edge))
		}
		name := sharedOutputs[shared]
		result.Diagrams = append(result.Diagrams, ClassDiagram{Name: name, Nodes: nodes, Relations: relations})
		result.Placements = append(result.Placements, sharedPlacement(name, shared))
	}
	return result, nil
}
