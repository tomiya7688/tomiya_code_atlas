package renderers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

var activityUnsafeID = regexp.MustCompile(`[^0-9A-Za-z_]`)

//	{
//	  責務: [RenderActivityFlowchart: logical activity flowをMermaid flowchartへ変換する]
//	  処理: [1: directionとnodeを順番に出力する; 2: decisionとactivityをshapeで区別する; 3: label付きedgeを出力し末尾改行を付ける]
//	  引数: [flow: nodeとedgeを持つlogical activity model; direction: Mermaid方向。空ならTD]
//	  戻り値: [string: 最終改行付きMermaid flowchart]
//	}
func RenderActivityFlowchart(flow generators.ActivityFlow, direction string) string {
	if direction == "" {
		direction = "TD"
	}
	lines := []string{fmt.Sprintf("flowchart %s", direction)}
	for _, node := range flow.Nodes {
		shape := "[" + node.Label + "]"
		if node.Kind == "decision" {
			shape = "{" + node.Label + "}"
		}
		lines = append(lines, "    "+activityNodeID(node.ID)+shape)
	}
	for _, edge := range flow.Edges {
		label := ""
		if edge.Label != "" {
			label = "|" + edge.Label + "|"
		}
		lines = append(lines, fmt.Sprintf("    %s -->%s %s", activityNodeID(edge.Source), label, activityNodeID(edge.Target)))
	}
	return strings.Join(lines, "\n") + "\n"
}

//	{
//	  責務: [activityNodeID: 利用できない文字をunderscoreへ置き換えてMermaid IDにする]
//	  処理: [1: prefix付きIDへ変換する; 2: 非対応文字をunderscoreへ置換する]
//	  引数: [id: 論理model上のnode ID]
//	  戻り値: [string: Mermaid内で使用するnode ID]
//	}
func activityNodeID(id string) string { return "act_" + activityUnsafeID.ReplaceAllString(id, "_") }
