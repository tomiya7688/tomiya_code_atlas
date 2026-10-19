package evaluators

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [DesignQualityThresholds: 設計評価で使う判定境界を保持する]
//	  フィールド: [HighFanIn: 情報findingを出す最小fan-in; HighFanOut: warningを出す最小fan-out; LargeSeries: 大規模seriesの最小node数; HighCrossSeriesRatio: 高結合候補の最小境界edge比率]
//	}
type DesignQualityThresholds struct {
	HighFanIn            int     `json:"high_fan_in"`
	HighFanOut           int     `json:"high_fan_out"`
	LargeSeries          int     `json:"large_series"`
	HighCrossSeriesRatio float64 `json:"high_cross_series_ratio"`
}

//	{
//	  責務: [DefaultDesignQualityThresholds: 設計評価の既定境界を返す]
//	  処理: [1: Python 1.0と同じ既定値を設定する]
//	  引数: [なし]
//	  戻り値: [DesignQualityThresholds: fan-in 3、fan-out 5、series 8、境界比率0.25の設定]
//	}
func DefaultDesignQualityThresholds() DesignQualityThresholds {
	return DesignQualityThresholds{HighFanIn: 3, HighFanOut: 5, LargeSeries: 8, HighCrossSeriesRatio: 0.25}
}

//	{
//	  責務: [EvaluateDesignQuality: Common IRと共通Analyzer結果からmetricと日本語findingを生成する]
//	  処理: [1: 閾値を検証する; 2: call/class graphとpartitionを得る; 3: metricと根拠付きfindingを安定順に作る]
//	  引数: [module: parser固有型を含まない解析済みCommon IR; thresholds: nilなら既定値]
//	  戻り値: [1: models.DesignQualityReport: rendererへ渡す受動的な評価結果; 2: error: 閾値またはgraphを処理できない理由]
//	}
func EvaluateDesignQuality(module commonir.Module, thresholds *DesignQualityThresholds) (models.DesignQualityReport, error) {
	selected := DefaultDesignQualityThresholds()
	if thresholds != nil {
		selected = *thresholds
	}
	if selected.HighFanIn < 1 || selected.HighFanOut < 1 || selected.LargeSeries < 1 {
		return models.DesignQualityReport{}, fmt.Errorf("設計評価の整数閾値は1以上にしてください")
	}
	if math.IsNaN(selected.HighCrossSeriesRatio) || selected.HighCrossSeriesRatio < 0 || selected.HighCrossSeriesRatio > 1 {
		return models.DesignQualityReport{}, fmt.Errorf("cross-series比率の閾値は0から1の範囲にしてください")
	}

	callSequence := analyzers.BuildSequenceRelationGraph(module, analyzers.ResolveCallSequences(module))
	callGraph, err := callSequence.RelationGraph()
	if err != nil {
		return models.DesignQualityReport{}, fmt.Errorf("呼び出しgraphを構築できません: %w", err)
	}
	classGraph := analyzers.BuildClassRelationGraph(module)
	classEdges := make([]analyzers.RelationEdge, 0, len(classGraph.Edges()))
	for _, edge := range classGraph.Edges() {
		classEdges = append(classEdges, analyzers.RelationEdge{Caller: edge.Caller, Callee: edge.Callee})
	}
	classRelations, err := analyzers.NewRelationGraph(classGraph.Nodes(), classEdges)
	if err != nil {
		return models.DesignQualityReport{}, fmt.Errorf("class graphを構築できません: %w", err)
	}
	callPartition, err := analyzers.PartitionGraph(callGraph, selected.HighFanIn, 3)
	if err != nil {
		return models.DesignQualityReport{}, fmt.Errorf("呼び出しgraphを分割できません: %w", err)
	}
	classPartition, err := analyzers.PartitionGraph(classRelations, selected.HighFanIn, 3)
	if err != nil {
		return models.DesignQualityReport{}, fmt.Errorf("class graphを分割できません: %w", err)
	}

	metrics := make([]models.DesignMetric, 0)
	metrics = appendGraphMetrics(metrics, callGraph, callPartition, "call_graph")
	metrics = appendGraphMetrics(metrics, classRelations, classPartition, "class_graph")
	findings := make([]models.DesignFinding, 0)
	findings = append(findings, hubFindings(callGraph.FanIn(), callGraph.FanOut(), "関数・メソッド", selected)...)
	findings = append(findings, hubFindings(classRelations.FanIn(), classRelations.FanOut(), "class", selected)...)
	findings = append(findings, cycleFindings(callGraph.Cycles(), "call-cycle", "呼び出しgraph")...)
	findings = append(findings, cycleFindings(classRelations.Cycles(), "class-cycle", "class graph")...)
	findings = append(findings, componentFindings(callGraph, "呼び出しgraph")...)
	findings = append(findings, componentFindings(classRelations, "class graph")...)
	findings = append(findings, partitionFindings(callPartition, "呼び出しgraph", selected)...)
	findings = append(findings, partitionFindings(classPartition, "class graph", selected)...)
	return models.DesignQualityReport{Metrics: metrics, Findings: findings}, nil
}

//	{
//	  責務: [appendGraphMetrics: graph規模、cycle、partitionのmetricを追加する]
//	  処理: [1: graph node/edge数を追加する; 2: cyclic SCCとpartition統計を追加する]
//	  引数: [metrics: 追記先; graph: 評価するgraph; partition: graphの分割結果; scope: metric対象名]
//	  戻り値: [追加済みmetric slice]
//	}
func appendGraphMetrics(metrics []models.DesignMetric, graph analyzers.RelationGraph, partition analyzers.GraphPartition, scope string) []models.DesignMetric {
	components := analyzers.CyclicStronglyConnectedComponents(graph)
	add := func(name string, value any) {
		metrics = append(metrics, models.DesignMetric{Name: name, Value: value, Scope: scope})
	}
	add("node_count", len(graph.Nodes()))
	add("edge_count", len(graph.Edges()))
	add("strongly_connected_component_count", len(components))
	largest := 0
	for _, component := range components {
		if len(component) > largest {
			largest = len(component)
		}
	}
	add("largest_scc_size", largest)
	add("cycle_count", partition.CycleCount)
	add("series_count", partition.SeriesCount())
	add("max_nodes_per_series", partition.MaxNodesPerSeries())
	add("avg_nodes_per_series", partition.AverageNodesPerSeries())
	add("max_partition_depth", partition.MaxDepth())
	add("shared_node_count", len(partition.Shared))
	add("shared_node_ratio", partition.SharedNodeRatio())
	add("cross_series_edge_count", partition.CrossSeriesEdgeCount)
	add("cross_series_edge_ratio", partition.CrossSeriesEdgeRatio())
	return metrics
}

//	{
//	  責務: [hubFindings: fan-in/outが閾値以上のnodeを説明付きで評価する]
//	  処理: [1: node名順にfan-in情報findingを追加する; 2: node名順にfan-out warningを追加する]
//	  引数: [fanIn: nodeごとの異なる呼び出し元数; fanOut: nodeごとの異なる呼び出し先数; scope: 対象nodeの種類; thresholds: 判定閾値]
//	  戻り値: [根拠と解釈を持つhub finding一覧]
//	}
func hubFindings(fanIn, fanOut map[string]int, scope string, thresholds DesignQualityThresholds) []models.DesignFinding {
	findings := make([]models.DesignFinding, 0)
	for _, name := range sortedCounts(fanIn) {
		count := fanIn[name]
		if count < thresholds.HighFanIn {
			continue
		}
		findings = append(findings, models.DesignFinding{Code: "high-fan-in", Severity: "info", Subject: name,
			Evidence:       fmt.Sprintf("%sのfan-in = %d", scope, count),
			Interpretation: "複数箇所から利用される依存hubです。",
			Note:           "logging、設定、event bus、dependency injection、共通error処理などは高fan-inでも妥当な場合があります。"})
	}
	for _, name := range sortedCounts(fanOut) {
		count := fanOut[name]
		if count < thresholds.HighFanOut {
			continue
		}
		findings = append(findings, models.DesignFinding{Code: "high-fan-out", Severity: "warning", Subject: name,
			Evidence:       fmt.Sprintf("%sのfan-out = %d", scope, count),
			Interpretation: "多くのnodeを調整または依存しており、責務が集中している可能性があります。",
			Note:           "大きな調整処理が意図的な場合もありますが、責務の集中を確認してください。"})
	}
	return findings
}

//	{
//	  責務: [cycleFindings: graph cycleを根拠付きwarningへ変換する]
//	  処理: [1: cycleをnode名列へ整形する; 2: cycle edge数と説明を記録する]
//	  引数: [cycles: 共通Analyzerが検出したcycle; code: finding識別子; scope: 対象graph名]
//	  戻り値: [cycle warning一覧]
//	}
func cycleFindings(cycles [][]string, code, scope string) []models.DesignFinding {
	findings := make([]models.DesignFinding, 0, len(cycles))
	for _, cycle := range cycles {
		edges := len(cycle) - 1
		if edges < 1 {
			edges = 1
		}
		findings = append(findings, models.DesignFinding{Code: code, Severity: "warning", Subject: strings.Join(cycle, " -> "),
			Evidence:       fmt.Sprintf("%d本のedgeを持つcycleを検出しました。", edges),
			Interpretation: fmt.Sprintf("%sに依存cycleがあります。", scope),
			Note:           "再帰処理が意図的な場合もあります。依存cycleの影響範囲を確認してください。"})
	}
	return findings
}

//	{
//	  責務: [componentFindings: cyclic SCCを相互到達領域のwarningへ変換する]
//	  処理: [1: cyclic SCCを取得する; 2: node名順のsubjectと到達数を記録する]
//	  引数: [graph: SCCを調べるrelation graph; scope: 対象graph名]
//	  戻り値: [cyclic SCC warning一覧]
//	}
func componentFindings(graph analyzers.RelationGraph, scope string) []models.DesignFinding {
	components := analyzers.CyclicStronglyConnectedComponents(graph)
	findings := make([]models.DesignFinding, 0, len(components))
	for _, component := range components {
		findings = append(findings, models.DesignFinding{Code: "strongly-connected-component", Severity: "warning", Subject: strings.Join(component, ", "),
			Evidence:       fmt.Sprintf("相互到達するnode数 = %d", len(component)),
			Interpretation: fmt.Sprintf("%sに単純な非循環層へ分けられない相互依存領域があります。", scope),
			Note:           "小さなSCCは意図的な再帰の場合があります。大きなSCCは変更影響が相互に伝わるため、設計上の境界を確認してください。"})
	}
	return findings
}

//	{
//	  責務: [partitionFindings: diagram partition統計から構造上の注意点を作る]
//	  処理: [1: 大規模seriesを検出する; 2: shared node集約を説明する; 3: series境界edge比率を判定する]
//	  引数: [partition: 共通Analyzerの分割結果; scope: 対象graph名; thresholds: seriesと結合比率の判定閾値]
//	  戻り値: [partitionに関するfinding一覧]
//	}
func partitionFindings(partition analyzers.GraphPartition, scope string, thresholds DesignQualityThresholds) []models.DesignFinding {
	findings := make([]models.DesignFinding, 0)
	if size := partition.MaxNodesPerSeries(); size >= thresholds.LargeSeries {
		findings = append(findings, models.DesignFinding{Code: "large-series", Severity: "warning", Subject: scope,
			Evidence:       fmt.Sprintf("最大partitionのnode数 = %d", size),
			Interpretation: "再帰分割後も大きな連結seriesが残っています。",
			Note:           "大きなまとまりが妥当な場合もあります。意図したarchitecture境界と比較してください。"})
	}
	if shared := len(partition.Shared); shared > 0 && partition.SeriesCount() > 0 {
		findings = append(findings, models.DesignFinding{Code: "shared-dependency-hubs", Severity: "info", Subject: scope,
			Evidence:       fmt.Sprintf("shared node = %d; regular series = %d; shared比率 = %.1f%%", shared, partition.SeriesCount(), partition.SharedNodeRatio()*100),
			Interpretation: "複数の関係がfan-inの高い共有nodeへ集約しています。",
			Note:           "これらのhubが意図した基盤か、偶発的な結合かを確認してください。"})
	}
	if count, ratio := partition.CrossSeriesEdgeCount, partition.CrossSeriesEdgeRatio(); count > 0 && ratio >= thresholds.HighCrossSeriesRatio {
		findings = append(findings, models.DesignFinding{Code: "cross-series-coupling", Severity: "warning", Subject: scope,
			Evidence:       fmt.Sprintf("series境界edge = %d; 比率 = %.1f%%", count, ratio*100),
			Interpretation: "依存の多くがdiagram seriesの境界を越えており、結合が高く局所性が弱い可能性があります。",
			Note:           "共有調整処理が複数seriesを結ぶ場合もあります。module/component境界と比較してください。"})
	}
	return findings
}

//	{
//	  責務: [sortedCounts: count mapのkeyを安定した辞書順で返す]
//	  処理: [1: map keyをsliceへ写す; 2: keyを辞書順に整列する]
//	  引数: [counts: node名をkeyとする集計map]
//	  戻り値: [辞書順node名一覧]
//	}
func sortedCounts(counts map[string]int) []string {
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
