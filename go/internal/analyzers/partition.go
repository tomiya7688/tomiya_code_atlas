package analyzers

import (
	"fmt"
	"sort"
)

// RelationEdge is a renderer-neutral directed relation used by graph analysis.
type RelationEdge struct{ Caller, Callee string }

// RelationGraph retains explicit isolated nodes and ordered relation facts.
type RelationGraph struct {
	nodes map[string]struct{}
	edges []RelationEdge
}

// NewRelationGraph validates and copies a generic relation graph.
func NewRelationGraph(nodes []string, edges []RelationEdge) (RelationGraph, error) {
	graph := RelationGraph{nodes: map[string]struct{}{}, edges: append([]RelationEdge(nil), edges...)}
	for _, node := range nodes {
		if node == "" {
			return RelationGraph{}, fmt.Errorf("graph node must not be empty")
		}
		graph.nodes[node] = struct{}{}
	}
	for i, e := range graph.edges {
		if e.Caller == "" || e.Callee == "" {
			return RelationGraph{}, fmt.Errorf("graph edge %d requires caller and callee", i)
		}
		graph.nodes[e.Caller] = struct{}{}
		graph.nodes[e.Callee] = struct{}{}
	}
	return graph, nil
}

// RelationGraphFromCallGraph adapts the call graph to shared graph algorithms.
func RelationGraphFromCallGraph(graph CallGraph) (RelationGraph, error) {
	edges := graph.Edges()
	out := make([]RelationEdge, 0, len(edges))
	for _, e := range edges {
		out = append(out, RelationEdge{e.Caller, e.Callee})
	}
	return NewRelationGraph(graph.Nodes(), out)
}

// Nodes returns all nodes lexically sorted.
func (graph RelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

// Edges returns relation facts in input order.
func (graph RelationGraph) Edges() []RelationEdge { return append([]RelationEdge(nil), graph.edges...) }

// FanIn counts distinct callers for every node.
func (graph RelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, toPairs(graph.edges))
}

// FanOut counts distinct callees for every node.
func (graph RelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, toPairs(graph.edges))
}

// Cycles returns each simple directed cycle once.
func (graph RelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, toPairs(graph.edges))
}

// GraphPartition contains deterministic caller-centered graph series.
type GraphPartition struct {
	Series               [][]string
	Shared               []string
	CrossSeriesEdgeCount int
	FanInDistribution    []int
	FanOutDistribution   []int
	CycleCount           int
	SeriesRoots          []string
	SeriesDepths         []int
	SeriesParents        []*string
	RegularEdgeCount     int
	TotalNodeCount       int
}

// PartitionGraph extracts high fan-in nodes and recursively separates closed branches.
func PartitionGraph(graph RelationGraph, fanInThreshold, minChildSeriesSize int) (GraphPartition, error) {
	if fanInThreshold < 1 {
		return GraphPartition{}, fmt.Errorf("fan-in threshold must be at least 1")
	}
	if minChildSeriesSize < 2 {
		return GraphPartition{}, fmt.Errorf("minimum child series size must be at least 2")
	}
	fanIn, fanOut := graph.FanIn(), graph.FanOut()
	sharedSet := map[string]struct{}{}
	regular := map[string]struct{}{}
	for n := range graph.nodes {
		if fanIn[n] >= fanInThreshold {
			sharedSet[n] = struct{}{}
		} else {
			regular[n] = struct{}{}
		}
	}
	shared := sortedKeys(sharedSet)
	regularEdges := make([]RelationEdge, 0)
	for _, e := range graph.edges {
		if _, ok := regular[e.Caller]; ok {
			if _, ok = regular[e.Callee]; ok {
				regularEdges = append(regularEdges, e)
			}
		}
	}
	cycles := graph.Cycles()
	cycleNodes := map[string]struct{}{}
	for _, cycle := range cycles {
		for index, node := range cycle {
			if index == len(cycle)-1 && len(cycle) > 1 && cycle[0] == node {
				continue
			}
			cycleNodes[node] = struct{}{}
		}
	}
	records := make([]partitionRecord, 0)
	for _, component := range weakComponents(regular, regularEdges) {
		records = append(records, splitComponent(component, regularEdges, cycleNodes, 0, nil, "", minChildSeriesSize)...)
	}
	result := GraphPartition{Shared: shared, CycleCount: len(cycles), RegularEdgeCount: len(regularEdges), TotalNodeCount: len(graph.nodes)}
	seriesByNode := map[string]int{}
	for i, r := range records {
		result.Series = append(result.Series, r.nodes)
		result.SeriesRoots = append(result.SeriesRoots, r.root)
		result.SeriesDepths = append(result.SeriesDepths, r.depth)
		result.SeriesParents = append(result.SeriesParents, r.parent)
		for _, n := range r.nodes {
			seriesByNode[n] = i
		}
	}
	for _, e := range regularEdges {
		if seriesByNode[e.Caller] != seriesByNode[e.Callee] {
			result.CrossSeriesEdgeCount++
		}
	}
	for _, n := range graph.Nodes() {
		result.FanInDistribution = append(result.FanInDistribution, fanIn[n])
		result.FanOutDistribution = append(result.FanOutDistribution, fanOut[n])
	}
	sort.Ints(result.FanInDistribution)
	sort.Ints(result.FanOutDistribution)
	return result, nil
}

func (partition GraphPartition) SeriesCount() int { return len(partition.Series) }
func (partition GraphPartition) MaxNodesPerSeries() int {
	n := 0
	for _, s := range partition.Series {
		if len(s) > n {
			n = len(s)
		}
	}
	return n
}
func (partition GraphPartition) MaxDepth() int {
	n := 0
	for _, d := range partition.SeriesDepths {
		if d > n {
			n = d
		}
	}
	return n
}
func (partition GraphPartition) AverageNodesPerSeries() float64 {
	if len(partition.Series) == 0 {
		return 0
	}
	total := 0
	for _, s := range partition.Series {
		total += len(s)
	}
	return float64(total) / float64(len(partition.Series))
}
func (partition GraphPartition) CrossSeriesEdgeRatio() float64 {
	if partition.RegularEdgeCount == 0 {
		return 0
	}
	return float64(partition.CrossSeriesEdgeCount) / float64(partition.RegularEdgeCount)
}
func (partition GraphPartition) SharedNodeRatio() float64 {
	if partition.TotalNodeCount == 0 {
		return 0
	}
	return float64(len(partition.Shared)) / float64(partition.TotalNodeCount)
}

type partitionRecord struct {
	nodes  []string
	root   string
	depth  int
	parent *string
}
type splitCandidate struct {
	distance      int
	parent, child string
	subtree       map[string]struct{}
}

func weakComponents(nodes map[string]struct{}, edges []RelationEdge) []map[string]struct{} {
	adj := map[string]map[string]struct{}{}
	for n := range nodes {
		adj[n] = map[string]struct{}{}
	}
	for _, e := range edges {
		adj[e.Caller][e.Callee] = struct{}{}
		adj[e.Callee][e.Caller] = struct{}{}
	}
	remaining := map[string]struct{}{}
	for n := range nodes {
		remaining[n] = struct{}{}
	}
	components := make([]map[string]struct{}, 0)
	for len(remaining) > 0 {
		starts := sortedKeys(remaining)
		start := starts[0]
		stack := []string{start}
		component := map[string]struct{}{}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, ok := remaining[n]; !ok {
				continue
			}
			delete(remaining, n)
			component[n] = struct{}{}
			neighbors := sortedKeys(adj[n])
			for i := len(neighbors) - 1; i >= 0; i-- {
				if _, ok := remaining[neighbors[i]]; ok {
					stack = append(stack, neighbors[i])
				}
			}
		}
		components = append(components, component)
	}
	return components
}

func splitComponent(component map[string]struct{}, edges []RelationEdge, cycleNodes map[string]struct{}, depth int, parentRoot *string, rootHint string, minSize int) []partitionRecord {
	internal := make([]RelationEdge, 0)
	outgoing := map[string]map[string]struct{}{}
	incoming := map[string]map[string]struct{}{}
	for n := range component {
		outgoing[n] = map[string]struct{}{}
		incoming[n] = map[string]struct{}{}
	}
	for _, e := range edges {
		if _, ok := component[e.Caller]; ok {
			if _, ok = component[e.Callee]; ok {
				internal = append(internal, e)
				outgoing[e.Caller][e.Callee] = struct{}{}
				incoming[e.Callee][e.Caller] = struct{}{}
			}
		}
	}
	roots := make([]string, 0)
	for n := range component {
		if len(incoming[n]) == 0 {
			roots = append(roots, n)
		}
	}
	sort.Strings(roots)
	root := rootHint
	if _, ok := component[root]; !ok {
		if len(roots) > 0 {
			root = roots[0]
		} else {
			root = sortedKeys(component)[0]
		}
	}
	distances := directedDistances(root, outgoing)
	candidates := make([]splitCandidate, 0)
	for _, p := range sortedKeys(component) {
		if len(outgoing[p]) < 2 {
			continue
		}
		for _, child := range sortedKeys(outgoing[p]) {
			subtree := reachable(child, outgoing)
			if len(subtree) < minSize || len(subtree) == len(component) {
				continue
			}
			hasCycle := false
			for n := range subtree {
				if _, ok := cycleNodes[n]; ok {
					hasCycle = true
					break
				}
			}
			if hasCycle {
				continue
			}
			boundaryIn, boundaryOut := make([]RelationEdge, 0), false
			for _, e := range internal {
				_, callerIn := subtree[e.Caller]
				_, calleeIn := subtree[e.Callee]
				if calleeIn && !callerIn {
					boundaryIn = append(boundaryIn, e)
				}
				if callerIn && !calleeIn {
					boundaryOut = true
				}
			}
			if boundaryOut || len(boundaryIn) != 1 || boundaryIn[0].Caller != p || boundaryIn[0].Callee != child {
				continue
			}
			d, ok := distances[p]
			if !ok {
				d = len(component) + 1
			}
			candidates = append(candidates, splitCandidate{d, p, child, subtree})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.distance != b.distance {
			return a.distance < b.distance
		}
		if a.parent != b.parent {
			return a.parent < b.parent
		}
		if a.child != b.child {
			return a.child < b.child
		}
		return len(a.subtree) > len(b.subtree)
	})
	selected := make([]splitCandidate, 0)
	occupied := map[string]struct{}{}
	for _, candidate := range candidates {
		overlap := false
		for n := range candidate.subtree {
			if _, ok := occupied[n]; ok {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		selected = append(selected, candidate)
		for n := range candidate.subtree {
			occupied[n] = struct{}{}
		}
	}
	local := map[string]struct{}{}
	for n := range component {
		if _, ok := occupied[n]; !ok {
			local[n] = struct{}{}
		}
	}
	if len(local) == 0 {
		local = component
		selected = nil
	}
	records := []partitionRecord{{nodes: sortedKeys(local), root: root, depth: depth, parent: parentRoot}}
	for _, child := range selected {
		parent := root
		records = append(records, splitComponent(child.subtree, internal, cycleNodes, depth+1, &parent, child.child, minSize)...)
	}
	return records
}
func directedDistances(root string, outgoing map[string]map[string]struct{}) map[string]int {
	dist := map[string]int{root: 0}
	queue := []string{root}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, child := range sortedKeys(outgoing[n]) {
			if _, ok := dist[child]; ok {
				continue
			}
			dist[child] = dist[n] + 1
			queue = append(queue, child)
		}
	}
	return dist
}
func reachable(start string, outgoing map[string]map[string]struct{}) map[string]struct{} {
	found := map[string]struct{}{}
	stack := []string{start}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := found[n]; ok {
			continue
		}
		found[n] = struct{}{}
		for _, child := range sortedKeys(outgoing[n]) {
			if _, ok := found[child]; !ok {
				stack = append(stack, child)
			}
		}
	}
	return found
}
func toPairs(edges []RelationEdge) []relationPair {
	out := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		out = append(out, relationPair{e.Caller, e.Callee})
	}
	return out
}

