// Package analyzersは、Common IRに対する決定的で言語非依存な解析を提供します。
package analyzers

import (
	"fmt"
	"sort"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [
//	    CallGraph: 呼び出し元・呼び出し先のnodeとdirect relationを保持する
//	  ]
//	  フィールド: [
//	    nodes: edgeの有無にかかわらず解析対象となる呼び出しnode
//	    edges: Common IRで観測した呼び出しrelation
//	  ]
//	}
type CallGraph struct {
	nodes map[string]struct{}
	edges []commonir.Relation
}

//	{
//	  責務: [
//	    BuildCallGraph: Common IRのfunctionとmethodから呼び出し関係を構築する
//	  ]
//	  処理: [
//	    1: functionとmethodの名前をcaller nodeとして登録する
//	    2: 各entityのcallsをdirect relationとして追加する
//	    3: runtime解決を推測せず、観測した名前のままgraphを返す
//	  ]
//	  引数: [
//	    module: function・methodと呼び出し名を含む解析済みCommon IR
//	  ]
//	  戻り値: [
//	    CallGraph: 解析した呼び出し元・呼び出し先nodeとrelation
//	  ]
//	}
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

//	{
//	  責務: [
//	    NewCallGraph: 入力nodeとrelationから検証済みgraphを作る
//	  ]
//	  処理: [
//	    1: node名とedge endpointが空でないことを検証する
//	    2: 空のCallTypeへdirectを設定する
//	    3: edge endpointもnodeへ登録してgraphを返す
//	  ]
//	  引数: [
//	    nodes: edgeがなくてもgraphに保持するnode名
//	    edges: 呼び出し元・呼び出し先を結ぶrelation
//	  ]
//	  戻り値: [
//	    1: CallGraph: 入力sliceから独立した検証済みgraph
//	    2: error: node名またはedge endpointが空の場合の理由
//	  ]
//	}
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

//	{
//	  責務: [
//	    CallGraph.Nodes: 孤立nodeを含む全node名を辞書順で返す
//	  ]
//	  処理: [
//	    1: node集合をsliceへコピーする
//	    2: コピーを辞書順に並べる
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: graph内の全node名を辞書順に並べたslice
//	  ]
//	}
func (graph CallGraph) Nodes() []string {
	nodes := make([]string, 0, len(graph.nodes))
	for node := range graph.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}

//	{
//	  責務: [
//	    CallGraph.Edges: Common IRで観測した順のrelationを返す
//	  ]
//	  処理: [
//	    1: 内部edgeをコピーし、呼び出し元による変更を分離する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []commonir.Relation: 入力順を保ったedgeの複製
//	  ]
//	}
func (graph CallGraph) Edges() []commonir.Relation {
	return append([]commonir.Relation(nil), graph.edges...)
}

//	{
//	  責務: [
//	    CallGraph.Outgoing: 指定callerから出るrelationを返す
//	  ]
//	  処理: [
//	    1: 全edgeからcallerが一致するものだけを抽出する
//	  ]
//	  引数: [
//	    caller: outgoing edgeを検索する呼び出し元node名
//	  ]
//	  戻り値: [
//	    []commonir.Relation: callerが一致するedgeを入力順に並べたslice
//	  ]
//	}
func (graph CallGraph) Outgoing(caller string) []commonir.Relation {
	edges := make([]commonir.Relation, 0)
	for _, edge := range graph.edges {
		if edge.Caller == caller {
			edges = append(edges, edge)
		}
	}
	return edges
}

//	{
//	  責務: [
//	    CallGraph.Incoming: 指定calleeへ入るrelationを返す
//	  ]
//	  処理: [
//	    1: 全edgeからcalleeが一致するものだけを抽出する
//	  ]
//	  引数: [
//	    callee: incoming edgeを検索する呼び出し先node名
//	  ]
//	  戻り値: [
//	    []commonir.Relation: calleeが一致するedgeを入力順に並べたslice
//	  ]
//	}
func (graph CallGraph) Incoming(callee string) []commonir.Relation {
	edges := make([]commonir.Relation, 0)
	for _, edge := range graph.edges {
		if edge.Callee == callee {
			edges = append(edges, edge)
		}
	}
	return edges
}

//	{
//	  責務: [
//	    CallGraph.FanIn: 各nodeを呼び出す異なるcallerの数を返す
//	  ]
//	  処理: [
//	    1: calleeごとにcallerを重複排除して集める
//	    2: callerがないnodeも0件として結果へ含める
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: node名から異なるcaller数への対応
//	  ]
//	}
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

//	{
//	  責務: [
//	    CallGraph.FanOut: 各nodeから呼び出される異なるcalleeの数を返す
//	  ]
//	  処理: [
//	    1: callerごとにcalleeを重複排除して集める
//	    2: 呼び出し先がないnodeも0件として結果へ含める
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: node名から異なるcallee数への対応
//	  ]
//	}
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

//	{
//	  責務: [
//	    CallGraph.HighFanInNodes: 指定件数以上のcallerを持つnodeを返す
//	  ]
//	  処理: [
//	    1: thresholdが1以上であることを検証する
//	    2: fan-inがthreshold以上のnodeを抽出する
//	    3: node名を辞書順に並べて返す
//	  ]
//	  引数: [
//	    threshold: nodeを対象とする最小の異なるcaller数。1以上
//	  ]
//	  戻り値: [
//	    1: []string: fan-inがthreshold以上のnode名を辞書順に並べたslice
//	    2: error: thresholdが1未満の場合の理由
//	  ]
//	}
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

//	{
//	  責務: [
//	    CallGraph.ReachableFrom: rootから到達できるnodeとrelationを深さ制限付きで返す
//	  ]
//	  処理: [
//	    1: maxDepthが負でないことを検証する
//	    2: 幅優先探索でrootからの最短深さを求める
//	    3: 深さ制限内のnodeとrelationから新しいgraphを作る
//	  ]
//	  引数: [
//	    root: 到達探索を開始するnode名
//	    maxDepth: rootから許可する最大edge数。nilは深さ制限なし
//	  ]
//	  戻り値: [
//	    1: CallGraph: rootから制限内で到達するnodeとedgeを含むgraph
//	    2: error: maxDepthが負、またはgraph生成時の入力が不正な場合の理由
//	  ]
//	}
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

//	{
//	  責務: [
//	    CallGraph.Cycles: 重複しない単純有向cycleを安定した順序で列挙する
//	  ]
//	  処理: [
//	    1: node順に探索を開始し、同じcycleの重複を避ける
//	    2: 各cycleを最小nodeから始まる形で列挙する
//	    3: cycle一覧を安定順に並べて返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    [][]string: cycleごとのnode順。各cycleは開始nodeを末尾にも含む
//	  ]
//	}
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

//	{
//	  責務: [
//	    qualifiedName: parentがあるentityの完全名を組み立てる
//	  ]
//	  処理: [
//	    1: parentがある場合はparentとentity名を連結する
//	    2: parentがなければentity名をそのまま返す
//	  ]
//	  引数: [
//	    entity: qualified nameを作るCommon IR entity
//	  ]
//	  戻り値: [
//	    string: parentがあればparent.name、なければentity.name
//	  ]
//	}
func qualifiedName(entity commonir.Entity) string {
	if entity.Parent != nil && *entity.Parent != "" {
		return *entity.Parent + "." + entity.Name
	}
	return entity.Name
}
