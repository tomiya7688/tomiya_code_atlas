// Package analyzers contains deterministic, language-neutral analyses over Common IR.
package analyzers

import (
	"fmt"
	"sort"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

// CallGraph is a renderer-neutral directed call relation graph.
type CallGraph struct {
	nodes map[string]struct{}
	edges []commonir.Relation
}

// BuildCallGraph derives direct call relations from function and method entities.
// Unresolved dynamic dispatch remains named as observed in Common IR; this
// analyzer does not guess import bindings or runtime behavior.
func BuildCallGraph(module commonir.Module) CallGraph {
	graph := CallGraph{nodes: make(map[string]struct{}), edges: make([]commonir.Relation, 0)}
	for _, entity := range module.Entities {
		if entity.Kind != "function" && entity.Kind != "method" {
			continue
		}
		caller := qualifiedName(entity)
		graph.nodes[caller] = struct{}{}
		for _, callee := range entity.Calls {
			graph.nodes[callee] = struct{}{}
			graph.edges = append(graph.edges, commonir.Relation{Caller: caller, Callee: callee, CallType: "direct"})
		}
	}
	return graph
}

// NewCallGraph creates a graph from logical facts and retains isolated nodes.
func NewCallGraph(nodes []string, edges []commonir.Relation) (CallGraph, error) {
	graph := CallGraph{nodes: make(map[string]struct{}), edges: append([]commonir.Relation(nil), edges...)}
	for _, node := range nodes {
		if node == "" {
			return CallGraph{}, fmt.Errorf("call graph node name must not be empty")
		}
		graph.nodes[node] = struct{}{}
	}
	for index, edge := range graph.edges {
		if edge.Caller == "" || edge.Callee == "" {
			return CallGraph{}, fmt.Errorf("call graph edge %d requires caller and callee", index)
		}
		if edge.CallType == "" {
			graph.edges[index].CallType = "direct"
		}
		graph.nodes[edge.Caller] = struct{}{}
		graph.nodes[edge.Callee] = struct{}{}
	}
	return graph, nil
}

// Nodes returns all declared and relation-referenced nodes in lexical order.
func (graph CallGraph) Nodes() []string {
	nodes := make([]string, 0, len(graph.nodes))
	for node := range graph.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}

// Edges returns relation facts in Common IR source order.
func (graph CallGraph) Edges() []commonir.Relation {
	return append([]commonir.Relation(nil), graph.edges...)
}

// Outgoing returns the source-ordered relations called by caller.
func (graph CallGraph) Outgoing(caller string) []commonir.Relation {
	edges := make([]commonir.Relation, 0)
	for _, edge := range graph.edges {
		if edge.Caller == caller {
			edges = append(edges, edge)
		}
	}
	return edges
}

// Incoming returns the source-ordered relations targeting callee.
func (graph CallGraph) Incoming(callee string) []commonir.Relation {
	edges := make([]commonir.Relation, 0)
	for _, edge := range graph.edges {
		if edge.Callee == callee {
			edges = append(edges, edge)
		}
	}
	return edges
}

// FanIn counts distinct callers for every node, including isolated nodes.
func (graph CallGraph) FanIn() map[string]int {
	callers := make(map[string]map[string]struct{}, len(graph.nodes))
	for _, edge := range graph.edges {
		if callers[edge.Callee] == nil {
			callers[edge.Callee] = make(map[string]struct{})
		}
		callers[edge.Callee][edge.Caller] = struct{}{}
	}
	counts := make(map[string]int, len(graph.nodes))
	for node := range graph.nodes {
		counts[node] = len(callers[node])
	}
	return counts
}

// FanOut counts distinct callees for every node, including isolated nodes.
func (graph CallGraph) FanOut() map[string]int {
	callees := make(map[string]map[string]struct{}, len(graph.nodes))
	for _, edge := range graph.edges {
		if callees[edge.Caller] == nil {
			callees[edge.Caller] = make(map[string]struct{})
		}
		callees[edge.Caller][edge.Callee] = struct{}{}
	}
	counts := make(map[string]int, len(graph.nodes))
	for node := range graph.nodes {
		counts[node] = len(callees[node])
	}
	return counts
}

// HighFanInNodes returns nodes with at least threshold distinct callers.
func (graph CallGraph) HighFanInNodes(threshold int) ([]string, error) {
	if threshold < 1 {
		return nil, fmt.Errorf("fan-in threshold must be at least 1")
	}
	counts := graph.FanIn()
	nodes := make([]string, 0)
	for node, count := range counts {
		if count >= threshold {
			nodes = append(nodes, node)
		}
	}
	sort.Strings(nodes)
	return nodes, nil
}

// ReachableFrom returns relations within maxDepth edges of root. A nil depth
// includes the complete reachable graph; depth zero includes only root.
func (graph CallGraph) ReachableFrom(root string, maxDepth *int) (CallGraph, error) {
	if maxDepth != nil && *maxDepth < 0 {
		return CallGraph{}, fmt.Errorf("max depth must be non-negative or nil")
	}
	adjacency := make(map[string][]string)
	for _, edge := range graph.edges {
		adjacency[edge.Caller] = append(adjacency[edge.Caller], edge.Callee)
	}
	depths := map[string]int{root: 0}
	queue := []string{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		depth := depths[node]
		if maxDepth != nil && depth >= *maxDepth {
			continue
		}
		for _, target := range adjacency[node] {
			nextDepth := depth + 1
			previous, visited := depths[target]
			if !visited || nextDepth < previous {
				depths[target] = nextDepth
				queue = append(queue, target)
			}
		}
	}
	selected := make([]commonir.Relation, 0)
	for _, edge := range graph.edges {
		depth, reachable := depths[edge.Caller]
		if reachable && (maxDepth == nil || depth < *maxDepth) {
			selected = append(selected, edge)
		}
	}
	nodes := make([]string, 0, len(depths))
	for node := range depths {
		nodes = append(nodes, node)
	}
	return NewCallGraph(nodes, selected)
}

// Cycles returns every simple directed cycle once, rotated to begin at its
// lexically smallest node and sorted lexically for stable output.
func (graph CallGraph) Cycles() [][]string {
	adjacency := make(map[string]map[string]struct{}, len(graph.nodes))
	for _, edge := range graph.edges {
		if adjacency[edge.Caller] == nil {
			adjacency[edge.Caller] = make(map[string]struct{})
		}
		adjacency[edge.Caller][edge.Callee] = struct{}{}
	}
	cycles := make([][]string, 0)
	for _, start := range graph.Nodes() {
		path := []string{start}
		visited := map[string]bool{start: true}
		var walk func(string)
		walk = func(node string) {
			next := make([]string, 0, len(adjacency[node]))
			for target := range adjacency[node] {
				if target >= start {
					next = append(next, target)
				}
			}
			sort.Strings(next)
			for _, target := range next {
				if target == start {
					cycles = append(cycles, append(append([]string(nil), path...), start))
					continue
				}
				if visited[target] {
					continue
				}
				visited[target] = true
				path = append(path, target)
				walk(target)
				path = path[:len(path)-1]
				delete(visited, target)
			}
		}
		walk(start)
	}
	sort.Slice(cycles, func(i, j int) bool { return fmt.Sprint(cycles[i]) < fmt.Sprint(cycles[j]) })
	return cycles
}

func qualifiedName(entity commonir.Entity) string {
	if entity.Parent != nil && *entity.Parent != "" {
		return *entity.Parent + "." + entity.Name
	}
	return entity.Name
}

