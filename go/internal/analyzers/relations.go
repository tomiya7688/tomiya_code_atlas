package analyzers

import (
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [
//	    ClassRelationEdge: class間の継承または解決済み利用関係を表す
//	  ]
//	  フィールド: [
//	    Caller: relation元のclass名
//	    Callee: relation先のclass名
//	    Relation: inheritanceまたはuses
//	  ]
//	}
type ClassRelationEdge struct{ Caller, Callee, Relation string }

//	{
//	  責務: [
//	    ClassRelationGraph: Common IRから解決したclass間relationを保持する
//	  ]
//	  フィールド: [
//	    nodes: project内classと、参照されたexternal base class
//	    edges: 継承と解決済みclass利用のrelation
//	  ]
//	}
type ClassRelationGraph struct {
	nodes map[string]struct{}
	edges []ClassRelationEdge
}

//	{
//	  責務: [
//	    BuildClassRelationGraph: Common IRからclass継承と解決済み利用relationを構築する
//	  ]
//	  処理: [
//	    1: class宣言を完全名と単純名で索引化する
//	    2: base class参照とmethod内callを既知classへ解決する
//	    3: external baseをnodeとして保ち、重複edgeを除いて返す
//	  ]
//	  引数: [
//	    module: class宣言、base、method callを持つCommon IR
//	  ]
//	  戻り値: [
//	    ClassRelationGraph: 既知class、external base、重複を除いたrelation
//	  ]
//	}
func BuildClassRelationGraph(module commonir.Module) ClassRelationGraph {
	// Class宣言を完全名と単純名のindexへ登録し、曖昧な単純名を識別します。
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
	// 完全名を優先し、部分名は一意に一致するclassだけ解決します。
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
	// base classはproject外でも参照事実としてnodeとedgeを保持します。
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
	// method callはowner classが既知で、calleeも一意に解決できる場合だけ追加します。
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

//	{
//	  責務: [
//	    ClassRelationGraph.Nodes: class relation graphのnode名を辞書順で返す
//	  ]
//	  処理: [
//	    1: graph内のclass名を辞書順に並べる
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: class relation graph内の辞書順node名
//	  ]
//	}
func (graph ClassRelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

//	{
//	  責務: [
//	    ClassRelationGraph.Edges: 重複を除いたclass relationをCommon IR順で返す
//	  ]
//	  処理: [
//	    1: 内部edgeをコピーして返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []ClassRelationEdge: 重複を除いたclass relationの複製
//	  ]
//	}
func (graph ClassRelationGraph) Edges() []ClassRelationEdge {
	return append([]ClassRelationEdge(nil), graph.edges...)
}

//	{
//	  責務: [
//	    ClassRelationGraph.FanIn: target classごとに異なるcaller数を返す
//	  ]
//	  処理: [
//	    1: class relationからtargetごとの異なるcaller数を集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: class名から異なるcaller数への対応
//	  ]
//	}
func (graph ClassRelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, classPairs(graph.edges))
}

//	{
//	  責務: [
//	    ClassRelationGraph.FanOut: source classごとに異なるcallee数を返す
//	  ]
//	  処理: [
//	    1: class relationからsourceごとの異なるcallee数を集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: class名から異なるcallee数への対応
//	  ]
//	}
func (graph ClassRelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, classPairs(graph.edges))
}

//	{
//	  責務: [
//	    ClassRelationGraph.Cycles: class relation graphの単純有向cycleを返す
//	  ]
//	  処理: [
//	    1: 共通cycle解析へnodeとrelationを渡す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    [][]string: class relation graphに含まれる単純有向cycle
//	  ]
//	}
func (graph ClassRelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, classPairs(graph.edges))
}

//	{
//	  責務: [
//	    ObjectRelationEdge: 静的に解決したobject参照とそのlabelを表す
//	  ]
//	  フィールド: [
//	    Caller: 参照を持つobject key
//	    Callee: 参照先object key
//	    Label: 参照の種類
//	  ]
//	}
type ObjectRelationEdge struct{ Caller, Callee, Label string }

//	{
//	  責務: [
//	    ObjectRelationGraph: scope付きobjectと解決済み参照を保持する
//	  ]
//	  フィールド: [
//	    nodes: scopeと名前から作ったobject key
//	    edges: 一意に解決できたobject参照
//	  ]
//	}
type ObjectRelationGraph struct {
	nodes map[string]struct{}
	edges []ObjectRelationEdge
}

//	{
//	  責務: [
//	    BuildObjectRelationGraph: Common IRのobject参照をscope内で解決する
//	  ]
//	  処理: [
//	    1: scopeと名前から各objectのkeyを作る
//	    2: 同scopeの参照を優先して解決する
//	    3: 全体で候補が一つの参照だけを採用し、重複edgeを除く
//	  ]
//	  引数: [
//	    module: objectと参照記録を持つCommon IR
//	  ]
//	  戻り値: [
//	    ObjectRelationGraph: scope付きobjectと一意に解決できた参照relation
//	  ]
//	}
func BuildObjectRelationGraph(module commonir.Module) ObjectRelationGraph {
	key := func(instance commonir.ObjectInstance) string {
		scope := "<module>"
		if instance.Scope != nil && *instance.Scope != "" {
			scope = *instance.Scope
		}
		return scope + ":" + instance.Name
	}
	// Scope付きkeyと名前別indexを作り、参照先を決定的に解決できるようにします。
	byKey := make(map[string]commonir.ObjectInstance, len(module.Objects))
	byScopeName := make(map[string]string, len(module.Objects))
	byName := make(map[string][]string)
	scopeToken := func(scope *string) string {
		if scope == nil {
			return "<nil>"
		}
		return "<value>" + *scope
	}
	// 同scopeの参照を優先し、scope内で見つからない名前は全体で一意な場合だけ結びます。
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

//	{
//	  責務: [
//	    ObjectRelationGraph.Nodes: object keyを辞書順で返す
//	  ]
//	  処理: [
//	    1: object keyを辞書順に並べる
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: scopeを含むobject keyの辞書順一覧
//	  ]
//	}
func (graph ObjectRelationGraph) Nodes() []string { return sortedKeys(graph.nodes) }

//	{
//	  責務: [
//	    ObjectRelationGraph.Edges: 重複を除いた参照edgeを入力順で返す
//	  ]
//	  処理: [
//	    1: 内部edgeをコピーして返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []ObjectRelationEdge: 重複を除いた参照edgeの複製
//	  ]
//	}
func (graph ObjectRelationGraph) Edges() []ObjectRelationEdge {
	return append([]ObjectRelationEdge(nil), graph.edges...)
}

//	{
//	  責務: [
//	    ObjectRelationGraph.FanIn: target objectごとに異なるsource object数を返す
//	  ]
//	  処理: [
//	    1: object relationからtargetごとの異なるcaller数を集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: object keyから異なるsource object数への対応
//	  ]
//	}
func (graph ObjectRelationGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, objectPairs(graph.edges))
}

//	{
//	  責務: [
//	    ObjectRelationGraph.FanOut: source objectごとに異なるtarget object数を返す
//	  ]
//	  処理: [
//	    1: object relationからsourceごとの異なるcallee数を集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: object keyから異なるtarget object数への対応
//	  ]
//	}
func (graph ObjectRelationGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, objectPairs(graph.edges))
}

//	{
//	  責務: [
//	    ObjectRelationGraph.Cycles: object参照graphの単純有向cycleを返す
//	  ]
//	  処理: [
//	    1: object relationを共通cycle解析へ渡す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    [][]string: object relation graphに含まれる単純有向cycle
//	  ]
//	}
func (graph ObjectRelationGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, objectPairs(graph.edges))
}

//	{
//	  責務: [
//	    ComponentDependencyUnit: project moduleと所属componentを結び付ける
//	  ]
//	  フィールド: [
//	    Module: module名とCommon IR import情報
//	    Component: moduleを集約するcomponent名
//	  ]
//	}
type ComponentDependencyUnit struct {
	Module    ModuleDependencyUnit
	Component string
}

//	{
//	  責務: [
//	    ComponentDependencyEdge: component間の一つの依存を表す
//	  ]
//	  フィールド: [
//	    Caller: 依存元component
//	    Callee: 依存先component
//	  ]
//	}
type ComponentDependencyEdge struct{ Caller, Callee string }

//	{
//	  責務: [
//	    ComponentDependencyGraph: internalとexternalのcomponent依存を保持する
//	  ]
//	  フィールド: [
//	    nodes: internalとexternalの全component名
//	    externalNodes: project外dependencyの識別名
//	  ]
//	}
type ComponentDependencyGraph struct {
	nodes, externalNodes map[string]struct{}
	edges                []ComponentDependencyEdge
}

//	{
//	  責務: [
//	    BuildComponentDependencyGraph: module importをcomponent単位に集約する
//	  ]
//	  処理: [
//	    1: module名から所属componentを索引化する
//	    2: project内importをcomponent relationに集約する
//	    3: project外packageをexternal nodeとして保持する
//	    4: 重複edgeと同一componentへのself relationを除く
//	  ]
//	  引数: [
//	    units: moduleと所属componentの対応、各moduleのimport情報
//	  ]
//	  戻り値: [
//	    ComponentDependencyGraph: internal/external nodeと集約済みcomponent relation
//	  ]
//	}
func BuildComponentDependencyGraph(units []ComponentDependencyUnit) ComponentDependencyGraph {
	// module名から所属componentを引けるindexを準備します。
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
	// 同一component内のedgeと重複relationは集約結果へ入れません。
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
	// project内importはcomponentへ集約し、外部importは明示したexternal nodeに残します。
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

//	{
//	  責務: [
//	    ComponentDependencyGraph.Nodes: internal・external nodeを辞書順で返す
//	  ]
//	  処理: [
//	    1: component nodeを辞書順に並べる
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: internalとexternalを含むcomponent node名
//	  ]
//	}
func (graph ComponentDependencyGraph) Nodes() []string { return sortedKeys(graph.nodes) }

//	{
//	  責務: [
//	    ComponentDependencyGraph.ExternalNodes: external dependency名を辞書順で返す
//	  ]
//	  処理: [
//	    1: external dependency nodeを辞書順に並べる
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []string: external dependency node名を辞書順に並べた一覧
//	  ]
//	}
func (graph ComponentDependencyGraph) ExternalNodes() []string {
	return sortedKeys(graph.externalNodes)
}

//	{
//	  責務: [
//	    ComponentDependencyGraph.Edges: 重複を除いたcomponent relationを返す
//	  ]
//	  処理: [
//	    1: 内部edgeをコピーして返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    []ComponentDependencyEdge: 重複を除いたcomponent edgeの複製
//	  ]
//	}
func (graph ComponentDependencyGraph) Edges() []ComponentDependencyEdge {
	return append([]ComponentDependencyEdge(nil), graph.edges...)
}

//	{
//	  責務: [
//	    ComponentDependencyGraph.FanIn: targetごとに異なる依存元component数を返す
//	  ]
//	  処理: [
//	    1: component relationからtargetごとの異なるcaller数を集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: component名から異なる依存元数への対応
//	  ]
//	}
func (graph ComponentDependencyGraph) FanIn() map[string]int {
	return relationFanIn(graph.nodes, componentPairs(graph.edges))
}

//	{
//	  責務: [
//	    ComponentDependencyGraph.FanOut: sourceごとに異なる依存先component数を返す
//	  ]
//	  処理: [
//	    1: component relationからsourceごとの異なるcallee数を集計する
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    map[string]int: component名から異なる依存先数への対応
//	  ]
//	}
func (graph ComponentDependencyGraph) FanOut() map[string]int {
	return relationFanOut(graph.nodes, componentPairs(graph.edges))
}

//	{
//	  責務: [
//	    ComponentDependencyGraph.Cycles: component依存の単純有向cycleを返す
//	  ]
//	  処理: [
//	    1: component dependencyを共通cycle解析へ渡す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    [][]string: component dependency graph内の単純有向cycle
//	  ]
//	}
func (graph ComponentDependencyGraph) Cycles() [][]string {
	return cyclesForRelations(graph.nodes, componentPairs(graph.edges))
}

//	{
//	  責務: [
//	    relationPair: labelを持たないrelationの元node・先nodeを組にする
//	  ]
//	  フィールド: [
//	    caller: relation元node
//	    callee: relation先node
//	  ]
//	}
type relationPair struct{ caller, callee string }

//	{
//	  責務: [
//	    classPairs: class relationから共通のnode pairを取り出す
//	  ]
//	  処理: [
//	    1: 各class edgeのCallerとCalleeをpairへコピーする
//	  ]
//	  引数: [
//	    edges: CallerとCalleeを持つclass relation一覧
//	  ]
//	  戻り値: [
//	    []relationPair: callerとcalleeを対応させたrelation一覧
//	  ]
//	}
func classPairs(edges []ClassRelationEdge) []relationPair {
	pairs := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		pairs = append(pairs, relationPair{e.Caller, e.Callee})
	}
	return pairs
}
//	{
//	  責務: [
//	    objectPairs: object relationから共通のnode pairを取り出す
//	  ]
//	  処理: [
//	    1: 各object edgeのCallerとCalleeをpairへコピーする
//	  ]
//	  引数: [
//	    edges: CallerとCalleeを持つobject reference一覧
//	  ]
//	  戻り値: [
//	    []relationPair: callerとcalleeを対応させたrelation一覧
//	  ]
//	}
func objectPairs(edges []ObjectRelationEdge) []relationPair {
	pairs := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		pairs = append(pairs, relationPair{e.Caller, e.Callee})
	}
	return pairs
}
//	{
//	  責務: [
//	    componentPairs: component relationから共通のnode pairを取り出す
//	  ]
//	  処理: [
//	    1: 各component edgeのCallerとCalleeをpairへコピーする
//	  ]
//	  引数: [
//	    edges: CallerとCalleeを持つcomponent dependency一覧
//	  ]
//	  戻り値: [
//	    []relationPair: callerとcalleeを対応させたrelation一覧
//	  ]
//	}
func componentPairs(edges []ComponentDependencyEdge) []relationPair {
	pairs := make([]relationPair, 0, len(edges))
	for _, e := range edges {
		pairs = append(pairs, relationPair{e.Caller, e.Callee})
	}
	return pairs
}
//	{
//	  責務: [
//	    relationFanIn: 各nodeを参照する異なるsource node数を集計する
//	  ]
//	  処理: [
//	    1: callee別に異なるcallerを集合へ集約する
//	    2: 全nodeについてcaller集合の要素数を返す
//	  ]
//	  引数: [
//	    nodes: 件数を出力する全node集合
//	    edges: 集計するsource・target relation
//	  ]
//	  戻り値: [
//	    map[string]int: targetごとの異なるsource数
//	  ]
//	}
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
//	{
//	  責務: [
//	    relationFanOut: 各nodeから参照する異なるtarget node数を集計する
//	  ]
//	  処理: [
//	    1: caller別に異なるcalleeを集合へ集約する
//	    2: 全nodeについてcallee集合の要素数を返す
//	  ]
//	  引数: [
//	    nodes: 件数を出力する全node集合
//	    edges: 集計するsource・target relation
//	  ]
//	  戻り値: [
//	    map[string]int: sourceごとの異なるtarget数
//	  ]
//	}
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
//	{
//	  責務: [
//	    cyclesForRelations: label付きrelationを共通cycle解析へ渡す
//	  ]
//	  処理: [
//	    1: relation pairを共通CallGraph形式へ変換する
//	    2: CallGraphのcycle解析を呼び出す
//	  ]
//	  引数: [
//	    nodes: cycle探索の対象node
//	    edges: cycle探索のrelation
//	  ]
//	  戻り値: [
//	    [][]string: node名の並びで表した単純有向cycle
//	  ]
//	}
func cyclesForRelations(nodes map[string]struct{}, edges []relationPair) [][]string {
	facts := make([]commonir.Relation, 0, len(edges))
	for _, e := range edges {
		facts = append(facts, commonir.Relation{Caller: e.caller, Callee: e.callee, CallType: "direct"})
	}
	graph, _ := NewCallGraph(sortedKeys(nodes), facts)
	return graph.Cycles()
}
//	{
//	  責務: [
//	    sortedKeys: setのkeyを辞書順に並べて返す
//	  ]
//	  処理: [
//	    1: setのkeyをsliceへコピーする
//	    2: keyを辞書順に並べて返す
//	  ]
//	  引数: [
//	    set: keyを取り出す集合
//	  ]
//	  戻り値: [
//	    []string: 重複のない辞書順key一覧
//	  ]
//	}
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
//	{
//	  責務: [
//	    simpleName: 修飾名から最後の区切り以降の名前を返す
//	  ]
//	  処理: [
//	    1: nameをdotで分割する
//	    2: 最後の要素を返す
//	  ]
//	  引数: [
//	    name: dotで区切られた修飾名
//	  ]
//	  戻り値: [
//	    string: 最後のdot以降の名前
//	  ]
//	}
func simpleName(name string) string { parts := strings.Split(name, "."); return parts[len(parts)-1] }
