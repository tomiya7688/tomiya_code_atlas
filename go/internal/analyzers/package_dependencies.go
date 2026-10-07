package analyzers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [
//	    ModuleDependencyUnit: project内moduleの識別情報とCommon IRを結び付ける
//	  ]
//	  フィールド: [
//	    Name: import解決に使うproject内のmodule名
//	    CommonIR: parserが抽出したlanguage-neutralなimport情報
//	    IsPackage: 相対importの基点がpackage自体かを表す
//	  ]
//	}
type ModuleDependencyUnit struct {
	Name      string
	CommonIR  commonir.Module
	IsPackage bool
}

//	{
//	  責務: [
//	    PackageDependencyGraph: project内moduleとその依存relationを保持する
//	  ]
//	  フィールド: [
//	    nodes: 依存の有無にかかわらず解析対象となるmodule名
//	    edges: project内で解決したmodule間の依存relation
//	  ]
//	}
type PackageDependencyGraph struct {
	nodes map[string]struct{}
	edges []PackageDependencyEdge
}

//	{
//	  責務: [
//	    PackageDependencyEdge: project内の一つのmodule依存を表す
//	  ]
//	  フィールド: [
//	    Caller: importを持つmodule名
//	    Callee: import先として解決されたmodule名
//	  ]
//	}
type PackageDependencyEdge struct{ Caller, Callee string }

//	{
//	  責務: [
//	    BuildPackageDependencyGraph: Common IRのimportをproject内moduleへ解決して依存graphを作る
//	  ]
//	  処理: [
//	    1: 全moduleを既知moduleとして登録する
//	    2: import先を既知moduleに対して解決する
//	    3: self relationと重複edgeを除いてgraphへ追加する
//	  ]
//	  引数: [
//	    units: project内module名、package情報、Common IRを対応させた一覧
//	  ]
//	  戻り値: [
//	    PackageDependencyGraph: 孤立moduleを含むnode一覧と重複なし依存edge
//	  ]
//	}
func BuildPackageDependencyGraph(units []ModuleDependencyUnit) PackageDependencyGraph {
	// 全moduleを先に索引化し、孤立moduleもgraph nodeとして保持します。
	known := make(map[string]ModuleDependencyUnit, len(units))
	graph := PackageDependencyGraph{nodes: make(map[string]struct{}, len(units)), edges: make([]PackageDependencyEdge, 0)}
	for _, unit := range units {
		known[unit.Name] = unit
		graph.nodes[unit.Name] = struct{}{}
	}
	// Importはproject内moduleへ解決し、self relationと重複edgeを除きます。
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

//	{
//	  責務: [
//	    ResolveModuleReference: absoluteまたはrelative importを既知のproject moduleへ解決する
//	  ]
//	  処理: [
//	    1: relative importは現在moduleとpackage階層から絶対名へ正規化する
//	    2: 完全一致、次に最長の既知prefixを探す
//	    3: project内で解決できない場合は空文字を返す
//	  ]
//	  引数: [
//	    unit: relative importの基準になるproject module
//	    reference: Common IRに記録されたimport名
//	    known: project内の全module。解決候補として使う
//	  ]
//	  戻り値: [
//	    string: 最も深く一致するmodule名。外部・未解決の場合は空文字
//	  ]
//	}
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

//	{
//	  責務: [
//	    PackageDependencyGraph.Edges: module依存relationを入力順で返す
//	  ]
//	  処理: [
//	    1: 内部edgeを複製し、呼び出し元の変更からgraphを保護する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []PackageDependencyEdge: project/importの入力順を保ったedgeの複製
//	  ]
//	}
func (graph PackageDependencyGraph) Edges() []PackageDependencyEdge {
	return append([]PackageDependencyEdge(nil), graph.edges...)
}

//	{
//	  責務: [
//	    PackageDependencyGraph.Nodes: project内の全module名を辞書順で返す
//	  ]
//	  処理: [
//	    1: node集合をsliceへ移し、辞書順に整列する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: 孤立nodeを含むmodule名を辞書順に並べた一覧
//	  ]
//	}
func (graph PackageDependencyGraph) Nodes() []string {
	nodes := make([]string, 0, len(graph.nodes))
	for node := range graph.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}

//	{
//	  責務: [
//	    PackageDependencyGraph.IsolatedNodes: incomingもoutgoingも持たないmoduleを返す
//	  ]
//	  処理: [
//	    1: 全edgeの両endpointを接続済みnodeとして記録する
//	    2: 接続されていないnodeを辞書順で返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: incoming/outgoing edgeを持たないmodule名の辞書順一覧
//	  ]
//	}
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

//	{
//	  責務: [
//	    PackageDependencyGraph.Cycles: 単純有向dependency cycleを安定順で返す
//	  ]
//	  処理: [
//	    1: module dependencyを共通call graph形式へ変換する
//	    2: 共通cycle解析を実行し、検証errorを呼び出し元へ返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    1: [][]string: node名の並びで表した単純dependency cycle
//	    2: error: 共通call graphへ変換できない場合の理由
//	  ]
//	}
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
