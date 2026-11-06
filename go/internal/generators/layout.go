package generators

import "github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"

//	{
//	  責務: [OutputPlacement: partition済みlogical outputの保存先metadataを保持する]
//	  フィールド: [OutputName: 出力名, RelativeDir: generator配下の相対directory, SeriesRoot: series root, ParentSeriesRoot: 親series root, Depth: 分割深さ, Shared: shared出力か]
//	}
type OutputPlacement struct {
	OutputName       string   `json:"output_name"`
	RelativeDir      []string `json:"relative_dir,omitempty"`
	SeriesRoot       string   `json:"series_root,omitempty"`
	ParentSeriesRoot *string  `json:"parent_series_root,omitempty"`
	Depth            int      `json:"depth,omitempty"`
	Shared           bool     `json:"shared,omitempty"`
}

//	{
//	  責務: [DiagramReference: 境界nodeから続きのdiagramへ辿る参照を保持する]
//	  フィールド: [
//	    Node: 境界node名,
//	    OutputName: 続きのdiagram名,
//	    Shared: 参照先がshared diagramであることを示す,
//	  ]
//	}
type DiagramReference struct {
	Node       string `json:"node"`
	OutputName string `json:"output_name"`
	Shared     bool   `json:"shared,omitempty"`
}

//	{
//	  責務: [partitionReferences: 通常seriesから境界先diagramへの参照一覧を作る]
//	  引数: [partition: 分割結果, seriesIndex: 対象series index, sharedOutputs: shared nodeからdiagram名への対応, seriesOutputs: regular nodeからdiagram名への対応]
//	  戻り値: [参照先diagram名を含む重複のない境界参照]
//	}
func partitionReferences(partition analyzers.GraphPartition, seriesIndex int, sharedOutputs map[string]string, seriesOutputs map[string]string) []DiagramReference {
	references := []DiagramReference{}
	for _, edge := range partition.BoundaryEdges {
		if edge.CallerSeries != nil && *edge.CallerSeries == seriesIndex {
			if edge.CalleeShared {
				references = append(references, DiagramReference{Node: edge.Relation.Callee, OutputName: sharedOutputs[edge.Relation.Callee], Shared: true})
			} else if edge.CalleeSeries != nil {
				references = append(references, DiagramReference{Node: edge.Relation.Callee, OutputName: seriesOutputs[edge.Relation.Callee]})
			}
		}
		if edge.CalleeSeries != nil && *edge.CalleeSeries == seriesIndex && edge.CallerShared {
			references = append(references, DiagramReference{Node: edge.Relation.Caller, OutputName: sharedOutputs[edge.Relation.Caller], Shared: true})
		}
	}
	return uniqueDiagramReferences(references)
}

//	{
//	  責務: [uniqueDiagramReferences: 同じnodeとdiagramへの参照を一件にまとめる]
//	  引数: [references: 重複を含むdiagram参照]
//	  戻り値: [入力順を保った有効な一意参照]
//	}
func uniqueDiagramReferences(references []DiagramReference) []DiagramReference {
	unique := make([]DiagramReference, 0, len(references))
	seen := map[DiagramReference]struct{}{}
	for _, reference := range references {
		if reference.OutputName == "" {
			continue
		}
		if _, exists := seen[reference]; exists {
			continue
		}
		seen[reference] = struct{}{}
		unique = append(unique, reference)
	}
	return unique
}

//	{
//	  責務: [filterDiagramReferences: 図に表示するparticipantの境界参照だけを残す]
//	  引数: [references: partition全体から作った境界参照, participants: 対象図に表示するnode]
//	  戻り値: [対象図に存在するnodeへの境界参照]
//	}
func filterDiagramReferences(references []DiagramReference, participants []string) []DiagramReference {
	visible := make([]DiagramReference, 0, len(references))
	for _, reference := range references {
		if containsString(participants, reference.Node) {
			visible = append(visible, reference)
		}
	}
	return visible
}

//	{
//	  責務: [sharedDiagramReferences: shared diagramから接続先regular seriesへの参照一覧を作る]
//	  引数: [sharedNode: shared diagramのnode名, partition: 分割結果, sharedOutputs: shared nodeからdiagram名への対応, seriesOutputs: regular nodeからdiagram名への対応]
//	  戻り値: [regular diagramを参照する一意なstub一覧]
//	}
func sharedDiagramReferences(sharedNode string, partition analyzers.GraphPartition, sharedOutputs, seriesOutputs map[string]string) []DiagramReference {
	references := []DiagramReference{}
	for _, edge := range partition.BoundaryEdges {
		if edge.CallerShared && edge.Relation.Caller == sharedNode {
			if edge.CalleeSeries != nil {
				references = append(references, DiagramReference{Node: edge.Relation.Callee, OutputName: seriesOutputs[edge.Relation.Callee]})
			} else if edge.CalleeShared && edge.Relation.Callee != sharedNode {
				references = append(references, DiagramReference{Node: edge.Relation.Callee, OutputName: sharedOutputs[edge.Relation.Callee], Shared: true})
			}
		}
		if edge.CalleeShared && edge.Relation.Callee == sharedNode {
			if edge.CallerSeries != nil {
				references = append(references, DiagramReference{Node: edge.Relation.Caller, OutputName: seriesOutputs[edge.Relation.Caller]})
			} else if edge.CallerShared && edge.Relation.Caller != sharedNode {
				references = append(references, DiagramReference{Node: edge.Relation.Caller, OutputName: sharedOutputs[edge.Relation.Caller], Shared: true})
			}
		}
	}
	return uniqueDiagramReferences(references)
}

//	{
//	  責務: [partitionStatistics: Analyzerのpartition結果をPython版と同じplain metricsへ変換する]
//	  引数: [partition: 共通Analyzerの分割結果]
//	  戻り値: [map[string]any: evaluatorや生成結果で共有できるmetrics]
//	}
func partitionStatistics(partition analyzers.GraphPartition) map[string]any {
	return map[string]any{
		"series_count":            partition.SeriesCount(),
		"max_nodes_per_series":    partition.MaxNodesPerSeries(),
		"avg_nodes_per_series":    partition.AverageNodesPerSeries(),
		"max_partition_depth":     partition.MaxDepth(),
		"cross_series_edge_count": partition.CrossSeriesEdgeCount,
		"cross_series_edge_ratio": partition.CrossSeriesEdgeRatio(),
		"shared_node_count":       len(partition.Shared),
		"shared_node_ratio":       partition.SharedNodeRatio(),
		"fan_in_distribution":     append([]int(nil), partition.FanInDistribution...),
		"fan_out_distribution":    append([]int(nil), partition.FanOutDistribution...),
		"cycle_count":             partition.CycleCount,
	}
}

//	{
//	  責務: [regularPlacement: 通常seriesのrenderer非依存な配置metadataを作る]
//	  引数: [outputName: 出力名, partition: 分割結果, seriesIndex: 対象seriesのindex]
//	  戻り値: [OutputPlacement: 安定directory名とseries階層metadata]
//	}
func regularPlacement(outputName string, partition analyzers.GraphPartition, seriesIndex int) OutputPlacement {
	root := partition.SeriesRoots[seriesIndex]
	return OutputPlacement{
		OutputName:       outputName,
		RelativeDir:      []string{stableOutputName("series", root, "series")},
		SeriesRoot:       root,
		ParentSeriesRoot: partition.SeriesParents[seriesIndex],
		Depth:            partition.SeriesDepths[seriesIndex],
	}
}

//	{
//	  責務: [sharedPlacement: 高fan-in shared outputの配置metadataを作る]
//	  引数: [outputName: 出力名, sharedRoot: shared node名]
//	  戻り値: [OutputPlacement: shared directoryとroot metadata]
//	}
func sharedPlacement(outputName, sharedRoot string) OutputPlacement {
	return OutputPlacement{
		OutputName:  outputName,
		RelativeDir: []string{"shared"},
		SeriesRoot:  sharedRoot,
		Shared:      true,
	}
}

func relationGraph(nodes []string, edges [][2]string) (analyzers.RelationGraph, error) {
	relations := make([]analyzers.RelationEdge, 0, len(edges))
	for _, edge := range edges {
		relations = append(relations, analyzers.RelationEdge{Caller: edge[0], Callee: edge[1]})
	}
	return analyzers.NewRelationGraph(nodes, relations)
}
