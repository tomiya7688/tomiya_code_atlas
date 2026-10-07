package analyzers

import (
	"fmt"
	"sort"
)

//	{
//	  責務: [
//	    RelationEdge: グラフ解析で使うcaller・calleeの有向relationを表す
//	  ]
//	  フィールド: [
//	    Caller: relation元node
//	    Callee: relation先node
//	  ]
//	}
type RelationEdge struct{ Caller, Callee string }

//	{
//	  責務: [
//	    RelationGraph: 孤立nodeと入力順を保ったrelationを保持する
//	  ]
//	  フィールド: [
//	    nodes: relationがないnodeを含む全node
//	    edges: 入力順を保つ有向relation
//	  ]
//	}
type RelationGraph struct {
	nodes map[string]struct{}
	edges []RelationEdge
}

//	{
//	  責務: [
//	    NewRelationGraph: nodeとedgeを検証して独立したRelationGraphを作る
//	  ]
//	  処理: [
//	    1: node名とedge endpointが空でないことを検証する
//	    2: edge endpointをnode集合へ加える
//	    3: edge sliceを複製してgraphを返す
//	  ]
//	  引数: [
//	    nodes: 明示的にgraphへ残すnode名。孤立nodeも含める
//	    edges: 有向graphを構成するcaller・callee relation
//	  ]
//	  戻り値: [
//	    1: RelationGraph: nodeとedge endpointを含む検証済みgraph
//	    2: error: 空node名または空edge endpointがある場合の理由
//	  ]
//	}
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

//	{
//	  責務: [
//	    RelationGraphFromCallGraph: CallGraphを共通relation graphへ変換する
//	  ]
//	  処理: [
//	    1: call relationから共通RelationEdgeを作る
//	    2: nodeとedgeをNewRelationGraphで検証する
//	  ]
//	  引数: [
//	    graph: nodeと呼び出しrelationを含むcall graph
//	  ]
//	  戻り値: [
//	    1: RelationGraph: call relationを一般化したgraph
//	    2: error: 入力に空node名またはendpointがある場合の理由
//	  ]
//	}
func RelationGraphFromCallGraph(graph CallGraph) (RelationGraph, error) {
	edges := graph.Edges()
	out := make([]RelationEdge, 0, len(edges))
	for _, e := range edges {
		out = append(out, RelationEdge{e.Caller, e.Callee})
	}
	return NewRelationGraph(graph.Nodes(), out)
}

//	{
//	  責務: [
//	    RelationGraph.Nodes: graph内のnode名を辞書順で返す
//	  ]
//	  処理: [
//	    1: node集合をsliceへ移して辞書順に並べる
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: graph nodeを辞書順に並べた一覧
//	  ]
//	}
func (graph RelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

//	{
//	  責務: [
//	    RelationGraph.Edges: 入力順のrelationを複製して返す
//	  ]
//	  処理: [
//	    1: 内部edge sliceを複製して返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []RelationEdge: 入力順を保ったrelationの複製
//	  ]
//	}
func (graph RelationGraph) Edges() []RelationEdge { return append([]RelationEdge(nil), graph.edges...) }

//	{
//	  責務: [
//	    RelationGraph.FanIn: 各nodeを呼び出す異なるcaller数を返す
//	  ]
//	  処理: [
//	    1: relationをcaller・callee pairへ変換して重複排除集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: node名から異なるcaller数への対応
//	  ]
//	}
func (graph RelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, toPairs(graph.edges))
}

//	{
//	  責務: [
//	    RelationGraph.FanOut: 各nodeから呼び出される異なるcallee数を返す
//	  ]
//	  処理: [
//	    1: relationをcaller・callee pairへ変換して重複排除集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: node名から異なるcallee数への対応
//	  ]
//	}
func (graph RelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, toPairs(graph.edges))
}

//	{
//	  責務: [
//	    RelationGraph.Cycles: relation graph内の単純有向cycleを返す
//	  ]
//	  処理: [
//	    1: 共通cycle解析へnodeとrelationを渡す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    [][]string: graph内の単純有向cycle。cycleごとにnode名を並べる
//	  ]
//	}
func (graph RelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, toPairs(graph.edges))
}

//	{
//	  責務: [
//	    GraphPartition: 関連nodeのseries分割結果と評価指標を保持する
//	  ]
//	  フィールド: [
//	    Series: 閉じた枝ごとのnode群
//	    Shared: fan-in threshold以上で共通groupへ分離したnode
//	    CrossSeriesEdgeCount: series境界をまたぐedge数
//	    FanInDistribution: nodeごとのfan-in値
//	    FanOutDistribution: nodeごとのfan-out値
//	    CycleCount: graph内のcycle数
//	    SeriesRoots: 各seriesのroot node
//	    SeriesDepths: rootからの分割深さ
//	    SeriesParents: 各seriesを分けた親root
//	    RegularEdgeCount: regular node間にあるedge数
//	    TotalNodeCount: graphのnode総数
//	  ]
//	}
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

//	{
//	  責務: [
//	    PartitionGraph: 高fan-in nodeを分離し、閉じた枝をseriesへ再帰分割する
//	  ]
//	  処理: [
//	    1: thresholdと最小枝サイズを検証する
//	    2: shared nodeとregular nodeを分ける
//	    3: regular graphのweak componentごとにcycleを避けて閉じた枝を分割する
//	    4: series間edgeとfan-in/out分布を集計して返す
//	  ]
//	  引数: [
//	    graph: nodeとrelationを保持する解析対象graph
//	    fanInThreshold: これ以上のfan-inを持つnodeをsharedへ分離する最小値
//	    minChildSeriesSize: subtreeを独立seriesにする最小node数
//	  ]
//	  戻り値: [
//	    1: GraphPartition: node分割とgraph指標
//	    2: error: fanInThresholdが1未満、またはminChildSeriesSizeが2未満の場合の理由
//	  ]
//	}
func PartitionGraph(graph RelationGraph, fanInThreshold, minChildSeriesSize int) (GraphPartition, error) {
	if fanInThreshold < 1 {
		return GraphPartition{}, fmt.Errorf("fan-in threshold must be at least 1")
	}
	if minChildSeriesSize < 2 {
		return GraphPartition{}, fmt.Errorf("minimum child series size must be at least 2")
	}
	fanIn, fanOut := graph.FanIn(), graph.FanOut()
	// 共有参照として扱うnodeを分け、枝分割の対象を限定します。
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
	// Shared nodeを通るedgeを除き、regular nodeだけのgraphを作ります。
	regularEdges := make([]RelationEdge, 0)
	for _, e := range graph.edges {
		if _, ok := regular[e.Caller]; ok {
			if _, ok = regular[e.Callee]; ok {
				regularEdges = append(regularEdges, e)
			}
		}
	}
	cycles := graph.Cycles()
	// Cycle内nodeは閉じた枝として切り出せないため、候補から除外します。
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
	// 独立したweak componentごとに、再帰分割のrootを選びます。
	for _, component := range weakComponents(regular, regularEdges) {
		records = append(records, splitComponent(component, regularEdges, cycleNodes, 0, nil, "", minChildSeriesSize)...)
	}
	result := GraphPartition{Shared: shared, CycleCount: len(cycles), RegularEdgeCount: len(regularEdges), TotalNodeCount: len(graph.nodes)}
	// 分割recordを出力用seriesへ移し、edgeの所属seriesを索引化します。
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
	// node単位のfan-in/out distributionを集計し、比較可能な順に揃えます。
	for _, n := range graph.Nodes() {
		result.FanInDistribution = append(result.FanInDistribution, fanIn[n])
		result.FanOutDistribution = append(result.FanOutDistribution, fanOut[n])
	}
	sort.Ints(result.FanInDistribution)
	sort.Ints(result.FanOutDistribution)
	return result, nil
}

//	{
//	  責務: [GraphPartition.SeriesCount: 分割後のseries数を返す]
//	  処理: [1: Series sliceの要素数を返す]
//	  引数: [なし]
//	  戻り値: [int: series数]
//	}
func (partition GraphPartition) SeriesCount() int { return len(partition.Series) }
//	{
//	  責務: [GraphPartition.MaxNodesPerSeries: 最も大きいseriesのnode数を返す]
//	  処理: [1: 各seriesのnode数を比較し最大値を求める]
//	  引数: [なし]
//	  戻り値: [int: 最大node数。seriesがない場合は0]
//	}
func (partition GraphPartition) MaxNodesPerSeries() int {
	n := 0
	for _, s := range partition.Series {
		if len(s) > n {
			n = len(s)
		}
	}
	return n
}
//	{
//	  責務: [GraphPartition.MaxDepth: seriesの最大分割深さを返す]
//	  処理: [1: SeriesDepthsの最大値を求める]
//	  引数: [なし]
//	  戻り値: [int: 最大深さ。seriesがない場合は0]
//	}
func (partition GraphPartition) MaxDepth() int {
	n := 0
	for _, d := range partition.SeriesDepths {
		if d > n {
			n = d
		}
	}
	return n
}
//	{
//	  責務: [GraphPartition.AverageNodesPerSeries: seriesあたりの平均node数を返す]
//	  処理: [1: 全seriesのnode数を合計しseries数で割る]
//	  引数: [なし]
//	  戻り値: [float64: 平均node数。seriesがない場合は0]
//	}
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
//	{
//	  責務: [GraphPartition.CrossSeriesEdgeRatio: regular edgeに占めるseries境界edgeの割合を返す]
//	  処理: [1: CrossSeriesEdgeCountをRegularEdgeCountで割る]
//	  引数: [なし]
//	  戻り値: [float64: 比率。regular edgeがない場合は0]
//	}
func (partition GraphPartition) CrossSeriesEdgeRatio() float64 {
	if partition.RegularEdgeCount == 0 {
		return 0
	}
	return float64(partition.CrossSeriesEdgeCount) / float64(partition.RegularEdgeCount)
}
//	{
//	  責務: [GraphPartition.SharedNodeRatio: 全nodeに占めるshared nodeの割合を返す]
//	  処理: [1: Shared node数をTotalNodeCountで割る]
//	  引数: [なし]
//	  戻り値: [float64: 比率。nodeがない場合は0]
//	}
func (partition GraphPartition) SharedNodeRatio() float64 {
	if partition.TotalNodeCount == 0 {
		return 0
	}
	return float64(len(partition.Shared)) / float64(partition.TotalNodeCount)
}

//	{
//	  責務: [partitionRecord: 一つのseriesと、その再帰分割上の位置を保持する]
//	  フィールド: [
//	    nodes: このseriesへ割り当てたnode
//	    root: seriesの起点node
//	    depth: 分割階層の深さ
//	    parent: このseriesを分けた親root。最上位ではnil
//	  ]
//	}
type partitionRecord struct {
	nodes  []string
	root   string
	depth  int
	parent *string
}
//	{
//	  責務: [splitCandidate: 閉じた枝としてseriesへ分離可能なsubtreeを保持する]
//	  フィールド: [
//	    distance: rootからparentまでの距離
//	    parent: subtreeへ入るedgeのcaller
//	    child: subtreeのroot node
//	    subtree: 分離候補に含まれる全node
//	  ]
//	}
type splitCandidate struct {
	distance      int
	parent, child string
	subtree       map[string]struct{}
}

//	{
//	  責務: [weakComponents: edgeの向きを無視した連結componentへnodeを分ける]
//	  処理: [
//	    1: edgeの両endpointを隣接nodeとして登録する
//	    2: 未訪問nodeから探索し、各componentを一度ずつ収集する
//	  ]
//	  引数: [
//	    nodes: componentへ分割するnode集合
//	    edges: 隣接nodeを決める有向relation。探索時は向きを無視する
//	  ]
//	  戻り値: [
//	    []map[string]struct{}: edgeの向きを無視した連結componentのnode集合
//	  ]
//	}
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

//	{
//	  責務: [splitComponent: 条件を満たす閉じた枝を一つのcomponentから再帰分割する]
//	  処理: [
//	    1: component内のroot、incoming/outgoing、rootからの距離を求める
//	    2: cycleがなく入口と出口が閉じたsubtreeだけを候補にする
//	    3: 重ならない候補を決定的な順序で選ぶ
//	    4: 残りnodeを親seriesにし、選んだsubtreeを再帰処理する
//	  ]
//	  引数: [
//	    component: 分割対象の連結node集合
//	    edges: component内で使うrelation一覧
//	    cycleNodes: cycleに含まれるため枝分割できないnode集合
//	    depth: 現在の再帰分割深さ
//	    parentRoot: 分割元seriesのroot。最上位ではnil
//	    rootHint: 再帰時に使うchild root候補
//	    minSize: 独立seriesとして許可する最小node数
//	  ]
//	  戻り値: [
//	    []partitionRecord: 現在のseriesと再帰分割したchild series
//	  ]
//	}
func splitComponent(component map[string]struct{}, edges []RelationEdge, cycleNodes map[string]struct{}, depth int, parentRoot *string, rootHint string, minSize int) []partitionRecord {
	// component内edgeから候補判定に使うincoming/outgoingと内部relationを作ります。
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
	// 再帰呼び出し時は指定rootを優先し、最上位ではsource rootを選びます。
	root := rootHint
	if _, ok := component[root]; !ok {
		if len(roots) > 0 {
			root = roots[0]
		} else {
			root = sortedKeys(component)[0]
		}
	}
	distances := directedDistances(root, outgoing)
	// cycle、外部入口、外部出口を含まず、単一edgeから入るsubtreeだけを候補にします。
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
	// 距離・node名で決めた順に、重ならない候補を選択します。
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
	// 残りnodeを現在のseriesに置き、選択したsubtreeごとにchild seriesを作ります。
	records := []partitionRecord{{nodes: sortedKeys(local), root: root, depth: depth, parent: parentRoot}}
	for _, child := range selected {
		parent := root
		records = append(records, splitComponent(child.subtree, internal, cycleNodes, depth+1, &parent, child.child, minSize)...)
	}
	return records
}
//	{
//	  責務: [directedDistances: directed edgeに沿ったrootからの最短距離を求める]
//	  処理: [1: 幅優先探索で各nodeの初回到達距離を記録する]
//	  引数: [
//	    root: 距離0となる開始node
//	    outgoing: caller nodeから直接callee nodeを引く隣接map
//	  ]
//	  戻り値: [
//	    map[string]int: 到達nodeからedge数で表した最短距離への対応
//	  ]
//	}
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
//	{
//	  責務: [reachable: directed edgeに沿ってstartから到達できるnodeを集める]
//	  処理: [1: stackで探索し、訪問済みnodeを再訪しない]
//	  引数: [
//	    start: 探索を開始するnode
//	    outgoing: caller nodeから直接callee nodeを引く隣接map
//	  ]
//	  戻り値: [
//	    map[string]struct{}: startから有向edgeに沿って到達できるnode集合
//	  ]
//	}
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
//	{
//	  責務: [toPairs: RelationEdgeを共通集計用のcaller・callee pairへ変換する]
//	  処理: [1: edge順を保ってendpoint pairを作る]
//	  引数: [
//	    edges: callerとcalleeを持つ有向relation一覧
//	  ]
//	  戻り値: [
//	    []relationPair: CallerとCalleeを保ったpair一覧
//	  ]
//	}
func toPairs(edges []RelationEdge) []relationPair {
	out := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		out = append(out, relationPair{e.Caller, e.Callee})
	}
	return out
}
