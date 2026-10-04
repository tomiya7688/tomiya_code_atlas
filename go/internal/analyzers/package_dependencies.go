package analyzers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

// ModuleDependencyUnit joins a project module identity with language-neutral
// import references already extracted into Common IR.
type ModuleDependencyUnit struct {
	Name      string
	CommonIR  commonir.Module
	IsPackage bool
}

// PackageDependencyGraph describes dependencies between modules in one project.
type PackageDependencyGraph struct {
	nodes map[string]struct{}
	edges []PackageDependencyEdge
}

// PackageDependencyEdge is a logical project-module dependency.
type PackageDependencyEdge struct{ Caller, Callee string }

// BuildPackageDependencyGraph resolves Common IR imports only against modules
// present in the project and omits self-relations and duplicate module pairs.
func BuildPackageDependencyGraph(units []ModuleDependencyUnit) PackageDependencyGraph {
	known := make(map[string]ModuleDependencyUnit, len(units))
	graph := PackageDependencyGraph{nodes: make(map[string]struct{}, len(units)), edges: make([]PackageDependencyEdge, 0)}
	for _, unit := range units {
		known[unit.Name] = unit
		graph.nodes[unit.Name] = struct{}{}
	}
	seen := make(map[string]struct{})
	for _, unit := range units {
		for _, reference := range unit.CommonIR.Imports {
			target := ResolveModuleReference(unit, reference, known)
			if target == "" || target == unit.Name {
				continue
			}
			key := unit.Name + "\x00" + target
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			graph.edges = append(graph.edges, PackageDependencyEdge{Caller: unit.Name, Callee: target})
		}
	}
	return graph
}

// ResolveModuleReference maps a normalized absolute or relative import to the
// deepest matching known project module. Empty means external/unresolved.
func ResolveModuleReference(unit ModuleDependencyUnit, reference string, known map[string]ModuleDependencyUnit) string {
	candidate := reference
	if strings.HasPrefix(reference, ".") {
		level := len(reference) - len(strings.TrimLeft(reference, "."))
		tail := reference[level:]
		base := strings.Split(unit.Name, ".")
		if !unit.IsPackage {
			if len(base) > 0 {
				base = base[:len(base)-1]
			} else {
				base = nil
			}
		}
		remove := level - 1
		if remove > 0 {
			if remove >= len(base) {
				base = nil
			} else {
				base = base[:len(base)-remove]
			}
		}
		if tail != "" {
			base = append(base, strings.Split(tail, ".")...)
		}
		candidate = strings.Join(base, ".")
	}
	if _, exists := known[candidate]; exists {
		return candidate
	}
	parts := strings.Split(candidate, ".")
	for end := len(parts) - 1; end > 0; end-- {
		prefix := strings.Join(parts[:end], ".")
		if _, exists := known[prefix]; exists {
			return prefix
		}
	}
	return ""
}

// Edges returns dependency relations in project and import order.
func (graph PackageDependencyGraph) Edges() []PackageDependencyEdge {
	return append([]PackageDependencyEdge(nil), graph.edges...)
}

// Nodes returns project modules in lexical order.
func (graph PackageDependencyGraph) Nodes() []string {
	nodes := make([]string, 0, len(graph.nodes))
	for node := range graph.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}

// IsolatedNodes returns modules that have no incoming or outgoing project edge.
func (graph PackageDependencyGraph) IsolatedNodes() []string {
	connected := make(map[string]struct{}, len(graph.edges)*2)
	for _, edge := range graph.edges {
		connected[edge.Caller] = struct{}{}
		connected[edge.Callee] = struct{}{}
	}
	isolated := make([]string, 0)
	for node := range graph.nodes {
		if _, exists := connected[node]; !exists {
			isolated = append(isolated, node)
		}
	}
	sort.Strings(isolated)
	return isolated
}

// Cycles returns deterministic simple dependency cycles.
func (graph PackageDependencyGraph) Cycles() ([][]string, error) {
	plainEdges := graph.Edges()
	callEdges := make([]commonir.Relation, len(plainEdges))
	for index, edge := range plainEdges {
		callEdges[index] = commonir.Relation{Caller: edge.Caller, Callee: edge.Callee, CallType: "direct"}
	}
	callGraph, err := NewCallGraph(graph.Nodes(), callEdges)
	if err != nil {
		return nil, fmt.Errorf("build package cycle graph: %w", err)
	}
	return callGraph.Cycles(), nil
}

