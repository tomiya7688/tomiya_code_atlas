package renderers

import (
	"encoding/hex"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

//	{
//	  責務: [RenderCIWorkflowMermaid: CI job graphをMermaid flowchartとして出力する]
//	  処理: [1: triggerとjob nodeをescapeして出力する, 2: 解析済みedgeを追加する, 3: 末尾改行を付けて返す]
//	  引数: [diagram: generatorが構築したCI workflow logical model]
//	  戻り値: [string: Mermaid flowchart text]
//	}
func RenderCIWorkflowMermaid(diagram generators.CIWorkflowDiagram) string {
	triggerLabel := strings.Join(diagram.Triggers, ", ")
	lines := []string{"flowchart LR", `    ci_trigger["` + escapeCILabel(triggerLabel) + `"]`}
	for _, job := range diagram.Jobs {
		lines = append(lines, `    `+ciNodeID(job.ID)+`["`+escapeCILabel(job.Label)+`"]`)
	}
	for _, edge := range diagram.Edges {
		source := "ci_trigger"
		if !edge.FromTrigger {
			source = ciNodeID(edge.Source)
		}
		lines = append(lines, "    "+source+" --> "+ciNodeID(edge.Target))
	}
	return strings.Join(lines, "\n") + "\n"
}

//	{
//	  責務: [ciNodeID: job IDをMermaid node IDとして安全な文字列へ変換する]
//	  引数: [jobID: workflow内のjob ID]
//	  戻り値: [string: ci_ prefix付きnode ID]
//	}
func ciNodeID(jobID string) string {
	return "ci_job_" + hex.EncodeToString([]byte(jobID))
}

//	{
//	  責務: [escapeCILabel: Mermaid labelの改行・引用符・HTML記号を安全にescapeする]
//	  引数: [label: triggerまたはjob表示名]
//	  戻り値: [string: 1行のMermaid label]
//	}
func escapeCILabel(label string) string {
	label = strings.ReplaceAll(label, "&", "&amp;")
	label = strings.ReplaceAll(label, `"`, "&quot;")
	label = strings.ReplaceAll(label, "<", "&lt;")
	label = strings.ReplaceAll(label, ">", "&gt;")
	label = strings.ReplaceAll(label, "\r", "")
	return strings.ReplaceAll(label, "\n", "<br/>")
}
