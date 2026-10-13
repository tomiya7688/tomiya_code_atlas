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
