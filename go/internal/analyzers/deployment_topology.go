package analyzers

import "sort"

//	{
//	  責務: [
//	    Confidence: deployment factの確認確度を表す
//	  ]
//	  フィールド: [
//	    ConfidenceUnknown: 入力から確認できない状態
//	    ConfidenceInferred: 入力情報から推定した状態
//	    ConfidenceConfirmed: 入力に明示され確認できた状態
//	  ]
//	}
type Confidence string

const (
	ConfidenceUnknown   Confidence = "unknown"
	ConfidenceInferred  Confidence = "inferred"
	ConfidenceConfirmed Confidence = "confirmed"
)

//	{
//	  責務: [
//	    DeploymentNode: 出力形式に依存しないdeployment entityを保持する
//	  ]
//	  フィールド: [
//	    ID: nodeを識別するkey
//	    Label: 利用者向けの表示名
//	    Kind: entityの種類
//	    Confidence: factの確認確度
//	    Environment: 配置環境
//	    Source: factの情報源
//	    Metadata: 補足属性
//	  ]
//	}
type DeploymentNode struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Kind        string            `json:"kind"`
	Confidence  Confidence        `json:"confidence"`
	Environment string            `json:"environment,omitempty"`
	Source      string            `json:"source,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

//	{
//	  責務: [
//	    DeploymentConnection: deployment node間の論理relationを保持する
//	  ]
//	  フィールド: [
//	    Source: 関係元node ID
//	    Target: 関係先node ID
//	    Relation: 関係の種類
//	    Confidence: relation factの確認確度
//	  ]
//	}
type DeploymentConnection struct {
	Source     string     `json:"source"`
	Target     string     `json:"target"`
	Relation   string     `json:"relation"`
	Confidence Confidence `json:"confidence"`
}

//	{
//	  責務: [
//	    DeploymentTopology: 正規化したdeployment factsを保持する
//	  ]
//	  フィールド: [
//	    Nodes: deployment entityの一覧
//	    Connections: entity間relationの一覧
//	  ]
//	}
type DeploymentTopology struct {
	Nodes       []DeploymentNode       `json:"nodes"`
	Connections []DeploymentConnection `json:"connections"`
}

//	{
//	  責務: [
//	    MergeDeploymentTopologies: 複数topologyの重複factを確度優先で統合する
//	  ]
//	  処理: [
//	    1: nodeとconnectionを一意keyで集約する
//	    2: 同じfactでは確認確度が高い値を選ぶ
//	    3: IDとrelation順に整列して返す
//	  ]
//	  引数: [
//	    topologies: 統合するdeployment topology
//	  ]
//	  戻り値: [
//	    DeploymentTopology: 重複を除いた統合結果
//	  ]
//	}
func MergeDeploymentTopologies(topologies ...DeploymentTopology) DeploymentTopology {
	nodes := make(map[string]DeploymentNode)
	connections := make(map[deploymentConnectionKey]DeploymentConnection)
	for _, topology := range topologies {
		for _, node := range topology.Nodes {
			current, exists := nodes[node.ID]
			if !exists || confidenceRank(node.Confidence) > confidenceRank(current.Confidence) {
				nodes[node.ID] = cloneDeploymentNode(node)
			}
		}
		for _, connection := range topology.Connections {
			key := deploymentConnectionKey{connection.Source, connection.Target, connection.Relation}
			current, exists := connections[key]
			if !exists || confidenceRank(connection.Confidence) > confidenceRank(current.Confidence) {
				connections[key] = connection
			}
		}
	}
	result := emptyDeploymentTopology()
	for _, node := range nodes {
		result.Nodes = append(result.Nodes, node)
	}
	for _, connection := range connections {
		result.Connections = append(result.Connections, connection)
	}
	sort.Slice(result.Nodes, func(i, j int) bool { return result.Nodes[i].ID < result.Nodes[j].ID })
	sort.Slice(result.Connections, func(i, j int) bool {
		left, right := result.Connections[i], result.Connections[j]
		if left.Source != right.Source {
			return left.Source < right.Source
		}
		if left.Target != right.Target {
			return left.Target < right.Target
		}
		return left.Relation < right.Relation
	})
	return result
}

//	{
//	  責務: [
//	    DeploymentTopologyFromComponentGraph: component依存graphを推定deployment factsへ変換する
//	  ]
//	  処理: [
//	    1: internal componentとexternal dependencyをnodeへ変換する
//	    2: dependency edgeをimports connectionへ変換する
//	    3: すべてのfactをinferredとして返す
//	  ]
//	  引数: [
//	    graph: component単位の依存graph
//	  ]
//	  戻り値: [
//	    DeploymentTopology: 推定したnodeとconnection
//	  ]
//	}
func DeploymentTopologyFromComponentGraph(graph ComponentDependencyGraph) DeploymentTopology {
	result := emptyDeploymentTopology()
	external := make(map[string]struct{})
	for _, name := range graph.ExternalNodes() {
		external[name] = struct{}{}
	}
	for _, name := range graph.Nodes() {
		if _, isExternal := external[name]; isExternal {
			label := name
			if len(name) >= len("external:") && name[:len("external:")] == "external:" {
				label = name[len("external:"):]
			}
			result.Nodes = append(result.Nodes, DeploymentNode{ID: name, Label: label, Kind: "external-dependency", Confidence: ConfidenceInferred, Source: "source imports"})
		} else {
			result.Nodes = append(result.Nodes, DeploymentNode{ID: "component:" + name, Label: name, Kind: "application-component", Confidence: ConfidenceInferred, Source: "source imports"})
		}
	}
	for _, edge := range graph.Edges() {
		source, target := "component:"+edge.Caller, "component:"+edge.Callee
		if _, isExternal := external[edge.Caller]; isExternal {
			source = edge.Caller
		}
		if _, isExternal := external[edge.Callee]; isExternal {
			target = edge.Callee
		}
		result.Connections = append(result.Connections, DeploymentConnection{Source: source, Target: target, Relation: "imports", Confidence: ConfidenceInferred})
	}
	return result
}

//	{
//	  責務: [
//	    deploymentConnectionKey: connectionをsource・target・relationの組で識別する
//	  ]
//	  フィールド: [
//	    source: 関係元node ID
//	    target: 関係先node ID
//	    relation: 関係の種類
//	  ]
//	}
type deploymentConnectionKey struct{ source, target, relation string }

//	{
//	  責務: [
//	    confidenceRank: 確度の優先順を数値で返す
//	  ]
//	  処理: [
//	    1: confirmed、inferred、unknownの順に順位を割り当てる
//	    2: 未知の値はunknown相当として返す
//	  ]
//	  引数: [
//	    confidence: 比較するfactの確度
//	  ]
//	  戻り値: [
//	    int: 比較用の順位
//	  ]
//	}
func confidenceRank(confidence Confidence) int {
	switch confidence {
	case ConfidenceConfirmed:
		return 2
	case ConfidenceInferred:
		return 1
	default:
		return 0
	}
}

//	{
//	  責務: [
//	    cloneDeploymentNode: nodeとそのmetadata mapを独立した値へ複製する
//	  ]
//	  処理: [
//	    1: node valueを複製する
//	    2: metadataがある場合は別mapへ複製する
//	  ]
//	  引数: [
//	    node: 複製元のdeployment node
//	  ]
//	  戻り値: [
//	    DeploymentNode: 入力と独立したmetadataを持つnode
//	  ]
//	}
func cloneDeploymentNode(node DeploymentNode) DeploymentNode {
	if node.Metadata != nil {
		metadata := make(map[string]string, len(node.Metadata))
		for key, value := range node.Metadata {
			metadata[key] = value
		}
		node.Metadata = metadata
	}
	return node
}
