package analyzers

import (
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

// ResolvedCall retains the original call spelling and deterministic target resolution.
type ResolvedCall struct {
	Raw    string `json:"raw"`
	Target string `json:"target"`
}

// SequenceRelationGraph is the internal callable graph used by partition analysis.
type SequenceRelationGraph struct {
	nodes map[string]struct{}
	edges []RelationEdge
}

// ResolveCallSequences resolves ordered calls to declarations where unambiguous.
func ResolveCallSequences(module commonir.Module) map[string][]ResolvedCall {
	qualified := map[string]commonir.Entity{}
	bySimple := map[string][]string{}
	for _, entity := range module.Entities {
		if entity.Kind != "function" && entity.Kind != "method" {
			continue
		}
		name := qualifiedName(entity)
		qualified[name] = entity
		bySimple[entity.Name] = append(bySimple[entity.Name], name)
	}
	resolved := make(map[string][]ResolvedCall, len(qualified))
	for name, entity := range qualified {
		ordered := entity.CallSequence
		if len(ordered) == 0 {
			ordered = entity.Calls
		}
		calls := make([]ResolvedCall, 0, len(ordered))
		for _, raw := range ordered {
			calls = append(calls, ResolvedCall{Raw: raw, Target: resolveCallable(raw, entity.Parent, qualified, bySimple)})
		}
		resolved[name] = calls
	}
	return resolved
}

// ResolveCallableReference resolves one call using exact, self/cls, then unique-name rules.
func ResolveCallableReference(module commonir.Module, raw string, parent *string) string {
	qualified := map[string]commonir.Entity{}
	bySimple := map[string][]string{}
	for _, entity := range module.Entities {
		if entity.Kind != "function" && entity.Kind != "method" {
			continue
		}
		name := qualifiedName(entity)
		qualified[name] = entity
		bySimple[entity.Name] = append(bySimple[entity.Name], name)
	}
	return resolveCallable(raw, parent, qualified, bySimple)
}

func resolveCallable(raw string, parent *string, qualified map[string]commonir.Entity, bySimple map[string][]string) string {
	if _, ok := qualified[raw]; ok {
		return raw
	}
	if parent != nil && (*parent) != "" && (strings.HasPrefix(raw, "self.") || strings.HasPrefix(raw, "cls.")) {
		candidate := *parent + "." + strings.SplitN(raw, ".", 2)[1]
		if _, ok := qualified[candidate]; ok {
			return candidate
		}
	}
	if !strings.Contains(raw, ".") {
		matches := bySimple[raw]
		if len(matches) == 1 {
			return matches[0]
		}
	}
	return raw
}

// BuildSequenceRelationGraph produces deduplicated edges among known callables.
func BuildSequenceRelationGraph(module commonir.Module, sequences map[string][]ResolvedCall) SequenceRelationGraph {
	if sequences == nil {
		sequences = ResolveCallSequences(module)
	}
	graph := SequenceRelationGraph{nodes: make(map[string]struct{}, len(sequences)), edges: make([]RelationEdge, 0)}
	for name := range sequences {
		graph.nodes[name] = struct{}{}
	}
	seen := map[RelationEdge]struct{}{}
	for _, entity := range module.Entities {
		if entity.Kind != "function" && entity.Kind != "method" {
			continue
		}
		caller := qualifiedName(entity)
		for _, call := range sequences[caller] {
			if _, internal := graph.nodes[call.Target]; !internal {
				continue
			}
			edge := RelationEdge{Caller: caller, Callee: call.Target}
			if _, exists := seen[edge]; exists {
				continue
			}
			seen[edge] = struct{}{}
			graph.edges = append(graph.edges, edge)
		}
	}
	return graph
}

// Nodes returns callable names in lexical order.
func (graph SequenceRelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

// Edges returns unique internal call relations.
func (graph SequenceRelationGraph) Edges() []RelationEdge {
	return append([]RelationEdge(nil), graph.edges...)
}

// FanIn counts distinct callers.
func (graph SequenceRelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, toPairs(graph.edges))
}

// FanOut counts distinct callees.
func (graph SequenceRelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, toPairs(graph.edges))
}

// Cycles returns simple call-sequence graph cycles.
func (graph SequenceRelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, toPairs(graph.edges))
}

// RelationGraph adapts the sequence graph to partition/metric analyzers.
func (graph SequenceRelationGraph) RelationGraph() (RelationGraph, error) {
	return NewRelationGraph(graph.Nodes(), graph.Edges())
}

