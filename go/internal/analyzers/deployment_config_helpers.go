package analyzers

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//	{
//	  責務: [
//	    decodeYAMLDocuments: 複数documentのYAMLを検証してmappingとして返します
//	  ]
//	  処理: [
//	    1: documentを順にdecodeする
//	    2: 空documentを省く
//	    3: 失敗時にsourceNameを含むerrorを返す
//	  ]
//	  引数: [
//	    source: YAML本文
//	    sourceName: error表示用の相対path
//	  ]
//	  戻り値: [
//	    documents: 空mappingを除いたYAML document
//	    error: 構文不正時の解析error
//	  ]
//	}
func decodeYAMLDocuments(source, sourceName string) ([]map[string]any, error) {
	decoder := yaml.NewDecoder(strings.NewReader(source))
	documents := make([]map[string]any, 0)
	for {
		var value any
		err := decoder.Decode(&value)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return documents, nil
			}
			return nil, fmt.Errorf("invalid YAML in %s: %w", sourceName, err)
		}
		if document, ok := value.(map[string]any); ok {
			documents = append(documents, document)
		}
	}
}

//	{
//	  責務: [
//	    emptyDeploymentTopology: 空のnode・relationをJSON arrayとして表します
//	  ]
//	  処理: [
//	    1: 空sliceを初期化する
//	    2: 空のtopologyを返す
//	  ]
//	  引数: [
//	    なし
//	  ]
//	  戻り値: [
//	    DeploymentTopology: 空配列を持つtopology
//	  ]
//	}
func emptyDeploymentTopology() DeploymentTopology {
	return DeploymentTopology{
		Nodes:       make([]DeploymentNode, 0),
		Connections: make([]DeploymentConnection, 0),
	}
}

//	{
//	  責務: [
//	    sortDeploymentTopology: deployment factsの順序を決定的に整えます
//	  ]
//	  処理: [
//	    1: nodeをID順に並べる
//	    2: connectionをsource・target・relation順に並べる
//	  ]
//	  引数: [
//	    topology: 並べ替えるdeployment facts
//	  ]
//	  戻り値: [
//	    DeploymentTopology: 安定順のdeployment facts
//	  ]
//	}
func sortDeploymentTopology(topology DeploymentTopology) DeploymentTopology {
	sort.SliceStable(topology.Nodes, func(i, j int) bool { return topology.Nodes[i].ID < topology.Nodes[j].ID })
	sort.SliceStable(topology.Connections, func(i, j int) bool {
		left, right := topology.Connections[i], topology.Connections[j]
		if left.Source != right.Source {
			return left.Source < right.Source
		}
		if left.Target != right.Target {
			return left.Target < right.Target
		}
		return left.Relation < right.Relation
	})
	return topology
}

//	{
//	  責務: [
//	    classifyDeploymentService: service名とimage名から限定的なservice種別を分類します
//	  ]
//	  処理: [
//	    1: 小文字化する
//	    2: 固定markerを順に照合する
//	    3: 該当しない場合serviceを返す
//	  ]
//	  引数: [
//	    name: Compose service名
//	    image: 宣言されたcontainer image
//	  ]
//	  戻り値: [
//	    string: database・cache・message-queue・serviceのいずれか
//	  ]
//	}
func classifyDeploymentService(name, image string) string {
	value := strings.ToLower(name + " " + image)
	for _, group := range []struct {
		kind    string
		markers []string
	}{
		{"database", []string{"postgres", "mysql", "mariadb", "mongo", "sqlite"}},
		{"cache", []string{"redis", "memcached"}},
		{"message-queue", []string{"rabbitmq", "kafka", "nats", "activemq"}},
	} {
		for _, marker := range group.markers {
			if strings.Contains(value, marker) {
				return group.kind
			}
		}
	}
	return "service"
}

//	{
//	  責務: [
//	    uniqueStrings: 先に現れた値を保って重複を除去します
//	  ]
//	  処理: [
//	    1: 値を順に確認する
//	    2: 初出値だけを結果へ追加する
//	  ]
//	  引数: [
//	    values: 重複を除く文字列列
//	  ]
//	  戻り値: [
//	    string列: 初出順で重複を除いた値
//	  ]
//	}
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

//	{
//	  責務: [
//	    yamlStringList: YAMLのscalar・sequence・mappingをPython版と同じstring列へ変換します
//	  ]
//	  処理: [
//	    1: sequenceは要素を変換する
//	    2: mappingはkeyを返す
//	    3: scalarは単一要素にする
//	  ]
//	  引数: [
//	    value: YAML decoderが返した値
//	  ]
//	  戻り値: [
//	    string列: YAML値を展開した文字列
//	  ]
//	}
func yamlStringList(value any) []string {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, yamlValueString(item))
		}
		return result
	case map[string]any:
		result := make([]string, 0, len(typed))
		for key := range typed {
			result = append(result, key)
		}
		return result
	case nil:
		return nil
	default:
		return []string{yamlValueString(value)}
	}
}

//	{
//	  責務: [
//	    yamlValueString: YAML値をPython版のmetadata文字列表現へ変換します
//	  ]
//	  処理: [
//	    1: scalarを同じ表記へ変換する
//	    2: containerは安定したPython形式で表す
//	  ]
//	  引数: [
//	    value: YAML decoderが返した値
//	  ]
//	  戻り値: [
//	    string: metadataに保持する文字列表現
//	  ]
//	}
func yamlValueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case string:
		return typed
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case map[string]any, []any:
		return formatContainerMetadata(typed)
	default:
		return fmt.Sprint(value)
	}
}

//	{
//	  責務: [
//	    formatContainerMetadata: mappingとsequenceを決定的なmetadata文字列にします
//	  ]
//	  処理: [
//	    1: mapping keyをsortする
//	    2: 子要素をPython表記に変換する
//	    3: container記号で囲む
//	  ]
//	  引数: [
//	    value: YAML mappingまたはsequence
//	  ]
//	  戻り値: [
//	    string: 安定したcontainer表現
//	  ]
//	}
func formatContainerMetadata(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		items := make([]string, 0, len(keys))
		for _, key := range keys {
			items = append(items, quoteMetadataString(key)+": "+formatMetadataValue(typed[key]))
		}
		return "{" + strings.Join(items, ", ") + "}"
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, formatMetadataValue(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return formatMetadataValue(value)
	}
}

//	{
//	  責務: [
//	    formatMetadataValue: container内の値を決定的なmetadata表記に変換します
//	  ]
//	  処理: [
//	    1: stringを引用する
//	    2: null・bool・nested containerを整形する
//	    3: numberを文字列にする
//	  ]
//	  引数: [
//	    value: container内のYAML値
//	  ]
//	  戻り値: [
//	    string: container内のmetadata文字列
//	  ]
//	}
func formatMetadataValue(value any) string {
	if text, ok := value.(string); ok {
		return quoteMetadataString(text)
	}
	switch typed := value.(type) {
	case nil:
		return "None"
	case bool:
		return yamlValueString(typed)
	case map[string]any, []any:
		return formatContainerMetadata(typed)
	default:
		return fmt.Sprint(value)
	}
}

//	{
//	  責務: [
//	    quoteMetadataString: stringをquoteとescapeで囲みます
//	  ]
//	  処理: [
//	    1: backslashをescapeする
//	    2: 選んだquoteと制御文字をescapeする
//	    3: quoteで囲む
//	  ]
//	  引数: [
//	    value: 引用する文字列
//	  ]
//	  戻り値: [
//	    string: container内で使うquoted metadata string
//	  ]
//	}
func quoteMetadataString(value string) string {
	quote := "'"
	if strings.Contains(value, "'") && !strings.Contains(value, `"`) {
		quote = `"`
	}
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, quote, "\\"+quote)
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	return quote + escaped + quote
}
