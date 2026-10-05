package analyzers

import (
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

// ClassRelationEdge records inheritance or a resolved class use.
type ClassRelationEdge struct{ Caller, Callee, Relation string }

// ClassRelationGraph contains known classes and externally named base classes.
type ClassRelationGraph struct {
	nodes map[string]struct{}
	edges []ClassRelationEdge
}

// BuildClassRelationGraph derives inheritance/use relations from Common IR.
func BuildClassRelationGraph(module commonir.Module) ClassRelationGraph {
	classes := make([]commonir.Entity, 0)
	known := make(map[string]commonir.Entity)
	simple := make(map[string][]string)
	for _, entity := range module.Entities {
		if entity.Kind != "class" {
			continue
		}
		classes = append(classes, entity)
		name := qualifiedName(entity)
		known[name] = entity
		simple[simpleName(name)] = append(simple[simpleName(name)], name)
	}
	resolve := func(reference string) string {
		if _, exists := known[reference]; exists {
			return reference
		}
		parts := strings.Split(reference, ".")
		for end := len(parts); end > 0; end-- {
			candidate := strings.Join(parts[:end], ".")
			if _, exists := known[candidate]; exists {
				return candidate
			}
			matches := simple[parts[end-1]]
			if len(matches) == 1 {
				return matches[0]
			}
		}
		return ""
	}
	graph := ClassRelationGraph{nodes: make(map[string]struct{}), edges: make([]ClassRelationEdge, 0)}
	for name := range known {
		graph.nodes[name] = struct{}{}
	}
	seen := make(map[ClassRelationEdge]struct{})
	add := func(source, target, relation string, includeExternal bool) {
		if target == "" || source == target {
			return
		}
		if includeExternal {
			graph.nodes[target] = struct{}{}
		}
		edge := ClassRelationEdge{Caller: source, Callee: target, Relation: relation}
		if _, exists := seen[edge]; exists {
			return
		}
		seen[edge] = struct{}{}
		graph.edges = append(graph.edges, edge)
	}
	for _, entity := range classes {
		source := qualifiedName(entity)
		for _, base := range entity.Bases {
			target := resolve(base)
			if target == "" {
				target = base
			}
			add(source, target, "inheritance", true)
		}
	}
	for _, entity := range module.Entities {
		if entity.Kind != "method" || entity.Parent == nil {
			continue
		}
		if _, ownerKnown := known[*entity.Parent]; !ownerKnown {
			continue
		}
		for _, call := range entity.Calls {
			if target := resolve(call); target != "" {
				add(*entity.Parent, target, "uses", false)
			}
		}
	}
	return graph
}

// Nodes returns class relation nodes in lexical order.
func (graph ClassRelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

// Edges returns deduplicated relations in Common IR traversal order.
func (graph ClassRelationGraph) Edges() []ClassRelationEdge {
	return append([]ClassRelationEdge(nil), graph.edges...)
}

// FanIn counts distinct related classes by target.
func (graph ClassRelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, classPairs(graph.edges))
}

// FanOut counts distinct related classes by source.
func (graph ClassRelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, classPairs(graph.edges))
}

// Cycles returns simple directed relation cycles.
func (graph ClassRelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, classPairs(graph.edges))
}

// ObjectRelationEdge records a labeled static object reference.
type ObjectRelationEdge struct{ Caller, Callee, Label string }

// ObjectRelationGraph contains scoped object instances and resolved references.
type ObjectRelationGraph struct {
	nodes map[string]struct{}
	edges []ObjectRelationEdge
}

// BuildObjectRelationGraph resolves references only when their target is unique.
func BuildObjectRelationGraph(module commonir.Module) ObjectRelationGraph {
	key := func(instance commonir.ObjectInstance) string {
		scope := "<module>"
		if instance.Scope != nil && *instance.Scope != "" {
			scope = *instance.Scope
		}
		return scope + ":" + instance.Name
	}
	byKey := make(map[string]commonir.ObjectInstance, len(module.Objects))
	byScopeName := make(map[string]string, len(module.Objects))
	byName := make(map[string][]string)
	scopeToken := func(scope *string) string {
		if scope == nil {
			return "<nil>"
		}
		return "<value>" + *scope
	}
	for _, instance := range module.Objects {
		objectID := key(instance)
		byKey[objectID] = instance
		byScopeName[scopeToken(instance.Scope)+"\x00"+instance.Name] = objectID
		byName[instance.Name] = append(byName[instance.Name], objectID)
	}
	graph := ObjectRelationGraph{nodes: make(map[string]struct{}, len(byKey)), edges: make([]ObjectRelationEdge, 0)}
	for node := range byKey {
		graph.nodes[node] = struct{}{}
	}
	seen := make(map[ObjectRelationEdge]struct{})
	for _, instance := range module.Objects {
		source := key(instance)
		for _, reference := range instance.References {
			if len(reference) < 2 {
				continue
			}
			label, rawTarget := reference[0], reference[1]
			target := byScopeName[scopeToken(instance.Scope)+"\x00"+rawTarget]
			if target == "" {
				candidates := byName[rawTarget]
				if len(candidates) == 1 {
					target = candidates[0]
				}
			}
			if target == "" {
				continue
			}
			edge := ObjectRelationEdge{Caller: source, Callee: target, Label: label}
			if _, exists := seen[edge]; exists {
				continue
			}
			seen[edge] = struct{}{}
			graph.edges = append(graph.edges, edge)
		}
	}
	return graph
}

// Nodes returns object keys in lexical order.
func (graph ObjectRelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

// Edges returns deduplicated references in instance/reference order.
func (graph ObjectRelationGraph) Edges() []ObjectRelationEdge {
	return append([]ObjectRelationEdge(nil), graph.edges...)
}

// FanIn counts distinct source objects by target.
func (graph ObjectRelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, objectPairs(graph.edges))
}

// FanOut counts distinct target objects by source.
func (graph ObjectRelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, objectPairs(graph.edges))
}

// Cycles returns simple directed reference cycles.
func (graph ObjectRelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, objectPairs(graph.edges))
}

// ComponentDependencyUnit associates a project module with its component.
type ComponentDependencyUnit struct {
	Module    ModuleDependencyUnit
	Component string
}

// ComponentDependencyEdge is a dependency between two components.
type ComponentDependencyEdge struct{ Caller, Callee string }

// ComponentDependencyGraph contains internal and external component nodes.
type ComponentDependencyGraph struct {
	nodes, externalNodes map[string]struct{}
	edges                []ComponentDependencyEdge
}

// BuildComponentDependencyGraph aggregates module imports, preserving external packages.
func BuildComponentDependencyGraph(units []ComponentDependencyUnit) ComponentDependencyGraph {
	known := make(map[string]ModuleDependencyUnit, len(units))
	componentByModule := make(map[string]string, len(units))
	components := make(map[string]struct{})
	for _, unit := range units {
		known[unit.Module.Name] = unit.Module
		componentByModule[unit.Module.Name] = unit.Component
		components[unit.Component] = struct{}{}
	}
	graph := ComponentDependencyGraph{nodes: make(map[string]struct{}), externalNodes: make(map[string]struct{}), edges: make([]ComponentDependencyEdge, 0)}
	for component := range components {
		graph.nodes[component] = struct{}{}
	}
	seen := make(map[ComponentDependencyEdge]struct{})
	add := func(source, target string) {
		if source == target {
			return
		}
		edge := ComponentDependencyEdge{Caller: source, Callee: target}
		if _, exists := seen[edge]; exists {
			return
		}
		seen[edge] = struct{}{}
		graph.edges = append(graph.edges, edge)
	}
	for _, unit := range units {
		for _, reference := range unit.Module.CommonIR.Imports {
			targetModule := ResolveModuleReference(unit.Module, reference, known)
			if targetModule != "" {
				add(unit.Component, componentByModule[targetModule])
				continue
			}
			if strings.HasPrefix(reference, ".") {
				continue
			}
			externalName := strings.SplitN(reference, ".", 2)[0]
			if externalName == "" {
				continue
			}
			if _, isInternalComponent := components[externalName]; isInternalComponent {
				continue
			}
			external := "external:" + externalName
			graph.externalNodes[external] = struct{}{}
			graph.nodes[external] = struct{}{}
			add(unit.Component, external)
		}
	}
	return graph
}

// Nodes returns component nodes in lexical order.
func (graph ComponentDependencyGraph) Nodes() []string { return sortedKeys(graph.nodes) }

// ExternalNodes returns external dependency labels in lexical order.
func (graph ComponentDependencyGraph) ExternalNodes() []string {
	return sortedKeys(graph.externalNodes)
}

// Edges returns deduplicated component edges in project/import order.
func (graph ComponentDependencyGraph) Edges() []ComponentDependencyEdge {
	return append([]ComponentDependencyEdge(nil), graph.edges...)
}

// FanIn counts distinct source components by target.
func (graph ComponentDependencyGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, componentPairs(graph.edges))
}

// FanOut counts distinct target components by source.
func (graph ComponentDependencyGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, componentPairs(graph.edges))
}

// Cycles returns simple directed component dependency cycles.
func (graph ComponentDependencyGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, componentPairs(graph.edges))
}

type relationPair struct{ caller, callee string }

func classPairs(edges []ClassRelationEdge) []relationPair {
	pairs := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		pairs = append(pairs, relationPair{e.Caller, e.Callee})
	}
	return pairs
}
func objectPairs(edges []ObjectRelationEdge) []relationPair {
	pairs := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		pairs = append(pairs, relationPair{e.Caller, e.Callee})
	}
	return pairs
}
func componentPairs(edges []ComponentDependencyEdge) []relationPair {
	pairs := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		pairs = append(pairs, relationPair{e.Caller, e.Callee})
	}
	return pairs
}
func relationFanIn(nodes map[string]struct{}, edges []relationPair) map[string]int {
	callers := map[string]map[string]struct{}{}
	for _, e := range edges {
		if callers[e.callee] == nil {
			callers[e.callee] = map[string]struct{}{}
		}
		callers[e.callee][e.caller] = struct{}{}
	}
	out := map[string]int{}
	for n := range nodes {
		out[n] = len(callers[n])
	}
	return out
}
func relationFanOut(nodes map[string]struct{}, edges []relationPair) map[string]int {
	callees := map[string]map[string]struct{}{}
	for _, e := range edges {
		if callees[e.caller] == nil {
			callees[e.caller] = map[string]struct{}{}
		}
		callees[e.caller][e.callee] = struct{}{}
	}
	out := map[string]int{}
	for n := range nodes {
		out[n] = len(callees[n])
	}
	return out
}
func cyclesForRelations(nodes map[string]struct{}, edges []relationPair) [][]string {
	facts := make([]commonir.Relation, 0, len(edges))
	for _, e := range edges {
		facts = append(facts, commonir.Relation{Caller: e.caller, Callee: e.callee, CallType: "direct"})
	}
	graph, _ := NewCallGraph(sortedKeys(nodes), facts)
	return graph.Cycles()
}
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func simpleName(name string) string { parts := strings.Split(name, "."); return parts[len(parts)-1] }

