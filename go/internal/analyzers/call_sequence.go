package analyzers

import (
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [
//	    ResolvedCall: 呼び出し元の記述と、決定的に解決した対象を保持する
//	  ]
//	  フィールド: [
//	    Raw: Common IRに記録された呼び出し名
//	    Target: 解決後の呼び出し先名。解決できない場合はRawと同じ名前
//	  ]
//	}
type ResolvedCall struct {
	Raw    string `json:"raw"`
	Target string `json:"target"`
}

//	{
//	  責務: [
//	    SequenceRelationGraph: call sequenceから作る内部callable graphを保持する
//	  ]
//	  フィールド: [
//	    nodes: sequence内のfunctionとmethod名
//	    edges: sequenceで観測した内部呼び出しrelation
//	  ]
//	}
type SequenceRelationGraph struct {
	nodes map[string]struct{}
	edges []RelationEdge
}

//	{
//	  責務: [
//	    ResolveCallSequences: Common IRの順序付き呼び出しを曖昧でない宣言へ解決する
//	  ]
//	  処理: [
//	    1: functionとmethodの完全名・単純名indexを作る
//	    2: CallSequenceが空の場合はCallsを使用する
//	    3: 呼び出し順を保って各対象を解決する
//	  ]
//	  引数: [
//	    module: function・methodと順序付き呼び出しを含む解析済みCommon IR
//	  ]
//	  戻り値: [
//	    map[string][]ResolvedCall: callerの完全名から元名・解決先を順番に対応させた一覧
//	  ]
//	}
func ResolveCallSequences(module commonir.Module) map[string][]ResolvedCall {
	// 後続の名前解決で毎回全entityを検索しないよう、functionとmethodを索引化します。
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
	// Common IRのsource orderを維持して、各callを個別に解決します。
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

//	{
//	  責務: [
//	    ResolveCallableReference: module内の宣言情報を使って一つの呼び出し先を解決する
//	  ]
//	  処理: [
//	    1: functionとmethodの完全名・単純名indexを作る
//	    2: 完全一致、self/cls、曖昧でない単純名の順で解決する
//	  ]
//	  引数: [
//	    module: 解決先となるfunction・method宣言を含むCommon IR
//	    raw: parserが記録した呼び出し先名
//	    parent: 呼び出し元methodのowner名。top-level functionではnil
//	  ]
//	  戻り値: [
//	    string: 解決した完全名。曖昧または未解決ならrawをそのまま返す
//	  ]
//	}
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

//	{
//	  責務: [
//	    resolveCallable: 呼び出し名を完全一致、owner、単純名の優先規則で解決する
//	  ]
//	  処理: [
//	    1: 完全名に一致した場合、その名前を返す
//	    2: self/cls参照は同じowner内の完全名を探す
//	    3: 単純名の候補が一件だけならその完全名を返す
//	    4: 曖昧または未解決の場合は元の名前を保持する
//	  ]
//	  引数: [
//	    raw: parserが記録した呼び出し先名
//	    parent: 呼び出し元methodのowner名
//	    qualified: 完全名から宣言entityへのindex
//	    bySimple: 単純名から候補完全名一覧へのindex
//	  ]
//	  戻り値: [
//	    string: 一意に解決した完全名。解決できない場合はraw
//	  ]
//	}
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

//	{
//	  責務: [
//	    BuildSequenceRelationGraph: 解決済みcall sequenceから内部relation graphを作る
//	  ]
//	  処理: [
//	    1: sequencesがnilならmoduleから解決する
//	    2: 呼び出し先が既知nodeのedgeだけを残す
//	    3: 重複edgeを除き、source orderを保って返す
//	  ]
//	  引数: [
//	    module: graph nodeの宣言情報を持つCommon IR
//	    sequences: caller名から呼び出し順を引けるresolved sequence。nilならmoduleから作る
//	  ]
//	  戻り値: [
//	    SequenceRelationGraph: module内callee間の重複なしrelation graph
//	  ]
//	}
func BuildSequenceRelationGraph(module commonir.Module, sequences map[string][]ResolvedCall) SequenceRelationGraph {
	if sequences == nil {
		sequences = ResolveCallSequences(module)
	}
	// sequenceに含まれるcallerをgraph nodeとして登録します。
	graph := SequenceRelationGraph{nodes: make(map[string]struct{}, len(sequences)), edges: make([]RelationEdge, 0)}
	for name := range sequences {
		graph.nodes[name] = struct{}{}
	}
	// sequence順に内部calleeへのedgeだけを追加し、複数回の呼び出しをrelation一件へまとめます。
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

//	{
//	  責務: [
//	    SequenceRelationGraph.Nodes: callable名を辞書順で返す
//	  ]
//	  処理: [
//	    1: graphのnode集合を辞書順に並べる
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: graph nodeの完全名を辞書順に並べた一覧
//	  ]
//	}
func (graph SequenceRelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

//	{
//	  責務: [
//	    SequenceRelationGraph.Edges: 内部call relationの複製を返す
//	  ]
//	  処理: [
//	    1: 内部edge sliceを複製して返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []RelationEdge: 呼び出し元の順を保った内部relationの複製
//	  ]
//	}
func (graph SequenceRelationGraph) Edges() []RelationEdge {
	return append([]RelationEdge(nil), graph.edges...)
}

//	{
//	  責務: [
//	    SequenceRelationGraph.FanIn: 各nodeを呼び出す異なるcaller数を返す
//	  ]
//	  処理: [
//	    1: edgeをcaller/callee pairに変換して重複排除集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: node名から異なるcaller数への対応
//	  ]
//	}
func (graph SequenceRelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, toPairs(graph.edges))
}

//	{
//	  責務: [
//	    SequenceRelationGraph.FanOut: 各nodeから呼び出される異なるcallee数を返す
//	  ]
//	  処理: [
//	    1: edgeをcaller/callee pairに変換して重複排除集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: node名から異なるcallee数への対応
//	  ]
//	}
func (graph SequenceRelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, toPairs(graph.edges))
}

//	{
//	  責務: [
//	    SequenceRelationGraph.Cycles: call sequence graphのsimple cycleを列挙する
//	  ]
//	  処理: [
//	    1: nodeとedgeを共通cycle解析へ渡す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    [][]string: cycleごとのnode名。各cycleは開始nodeを末尾にも含む
//	  ]
//	}
func (graph SequenceRelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, toPairs(graph.edges))
}

//	{
//	  責務: [
//	    SequenceRelationGraph.RelationGraph: 共通partition・metric解析用のgraphへ変換する
//	  ]
//	  処理: [
//	    1: nodeとedgeを検証付き共通RelationGraphへ渡す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    1: RelationGraph型の結果
//	    2: error型の結果
//	  ]
//	}
func (graph SequenceRelationGraph) RelationGraph() (RelationGraph, error) {
	return NewRelationGraph(graph.Nodes(), graph.Edges())
}
