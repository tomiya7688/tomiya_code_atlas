package generators

//	{
//	  責務: [ActivityNode: language-neutralなactivity図nodeを保持する]
//	  フィールド: [ID: edgeが参照する安定ID; Label: 表示文字列; Kind: activityまたはdecision]
//	}
type ActivityNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

//	{
//	  責務: [ActivityEdge: activity図node間の遷移を保持する]
//	  フィールド: [Source: 遷移元ID; Target: 遷移先ID; Label: 任意の条件表示]
//	}
type ActivityEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

//	{
//	  責務: [ActivityFlow: 言語解析後のactivity図logical modelを保持する]
//	  フィールド: [Entry: entry node ID; Nodes: 表示順node; Edges: 表示順edge]
//	}
type ActivityFlow struct {
	Entry string         `json:"entry"`
	Nodes []ActivityNode `json:"nodes"`
	Edges []ActivityEdge `json:"edges"`
}

//	{
//	  責務: [GenerateActivityFlowchart: 言語adapterが生成したlogical flowを共通出力型として返す]
//	  処理: [1: 受け取ったflow modelをそのまま返す]
//	  引数: [flow: 言語解析後に正規化したactivity flow]
//	  戻り値: [ActivityFlow: rendererが利用するlanguage-neutral model]
//	}
func GenerateActivityFlowchart(flow ActivityFlow) ActivityFlow { return flow }
