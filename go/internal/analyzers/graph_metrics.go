package analyzers

import "sort"

//	{
//	  責務: [
//	    StronglyConnectedComponents: 有向graphを強連結componentへ分けて安定順で返す
//	  ]
//	  処理: [
//	    1: adjacencyを重複除去して辞書順に整える
//	    2: Tarjan法でnodeをstrongly connected componentへ分類する
//	    3: 各componentとcomponent一覧を辞書順に整列する
//	  ]
//	  引数: [
//	    graph: 孤立nodeを含む解析対象の有向relation graph
//	  ]
//	  戻り値: [
//	    [][]string: node名を辞書順にしたstrongly connected component一覧
//	  ]
//	}
func StronglyConnectedComponents(graph RelationGraph) [][]string {
	// 隣接先を集合化してから辞書順へ整え、探索順による出力差を防ぎます。
	adjacency := make(map[string][]string, len(graph.nodes))
	targets := make(map[string]map[string]struct{}, len(graph.nodes))
	for node := range graph.nodes {
		targets[node] = map[string]struct{}{}
	}
	for _, edge := range graph.edges {
		if targets[edge.Caller] == nil {
			targets[edge.Caller] = map[string]struct{}{}
		}
		if targets[edge.Callee] == nil {
			targets[edge.Callee] = map[string]struct{}{}
		}
		targets[edge.Caller][edge.Callee] = struct{}{}
	}
	for node, items := range targets {
		adjacency[node] = sortedKeys(items)
	}
	// Tarjan indexとlow-linkでstack上のstrongly connected componentを抽出します。
	index := 0
	indices := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	stack := make([]string, 0)
	components := make([][]string, 0)
	var visit func(string)
	visit = func(node string) {
		indices[node] = index
		low[node] = index
		index++
		stack = append(stack, node)
		onStack[node] = true
		for _, target := range adjacency[node] {
			if _, seen := indices[target]; !seen {
				visit(target)
				if low[target] < low[node] {
					low[node] = low[target]
				}
			} else if onStack[target] && indices[target] < low[node] {
				low[node] = indices[target]
			}
		}
		if low[node] != indices[node] {
			return
		}
		component := make([]string, 0)
		for len(stack) > 0 {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == node {
				break
			}
		}
		sort.Strings(component)
		components = append(components, component)
	}
	nodes := make([]string, 0, len(targets))
	for node := range targets {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if _, seen := indices[node]; !seen {
			visit(node)
		}
	}
	// 各componentとcomponent一覧を安定順にして、呼び出しごとの差をなくします。
	sort.Slice(components, func(i, j int) bool {
		a, b := components[i], components[j]
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		for k := range a {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return false
	})
	return components
}

//	{
//	  責務: [
//	    CyclicStronglyConnectedComponents: cycleを含むstrongly connected componentだけを返す
//	  ]
//	  処理: [
//	    1: graph全体のstrongly connected componentを求める
//	    2: 2 node以上のcomponentとself-loopを持つ1 node componentを残す
//	  ]
//	  引数: [
//	    graph: cycleの有無を調べる有向relation graph
//	  ]
//	  戻り値: [
//	    [][]string: cycleを含むnode名一覧
//	  ]
//	}
func CyclicStronglyConnectedComponents(graph RelationGraph) [][]string {
	selfLoops := map[string]struct{}{}
	for _, edge := range graph.edges {
		if edge.Caller == edge.Callee {
			selfLoops[edge.Caller] = struct{}{}
		}
	}
	out := make([][]string, 0)
	for _, component := range StronglyConnectedComponents(graph) {
		if len(component) > 1 {
			out = append(out, component)
			continue
		}
		if len(component) == 1 {
			if _, ok := selfLoops[component[0]]; ok {
				out = append(out, component)
			}
		}
	}
	return out
}
