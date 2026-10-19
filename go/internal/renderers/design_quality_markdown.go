package renderers

import (
	"fmt"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [RenderDesignQualityMarkdown: 設計品質reportを日本語Markdownへ直列化する]
//	  処理: [1: metric tableを出力する; 2: findingごとに根拠・解釈・補足を出力する; 3: 最終改行を付ける]
//	  引数: [report: 言語解析やrenderer構文を含まない設計品質結果]
//	  戻り値: [string: 人が読める日本語Markdown]
//	}
func RenderDesignQualityMarkdown(report models.DesignQualityReport) string {
	var output strings.Builder
	output.WriteString("# 設計品質レポート\n\n")
	output.WriteString("## 指標\n\n| 対象 | 指標 | 値 |\n| --- | --- | ---: |\n")
	for _, metric := range report.Metrics {
		fmt.Fprintf(&output, "| %s | %s | %s |\n", markdownCell(metric.Scope), markdownCell(metric.Name), markdownCell(fmt.Sprint(metric.Value)))
	}
	output.WriteString("\n## 指摘\n")
	if len(report.Findings) == 0 {
		output.WriteString("\n現在の閾値では構造上の指摘は見つかりませんでした。\n")
		return output.String()
	}
	for _, finding := range report.Findings {
		fmt.Fprintf(&output, "\n### [%s] %s: %s\n\n", strings.ToUpper(finding.Severity), markdownCell(finding.Code), markdownCell(finding.Subject))
		fmt.Fprintf(&output, "- 根拠: %s\n", markdownCell(finding.Evidence))
		fmt.Fprintf(&output, "- 解釈: %s\n", markdownCell(finding.Interpretation))
		if finding.Note != "" {
			fmt.Fprintf(&output, "- 補足: %s\n", markdownCell(finding.Note))
		}
	}
	return output.String()
}

//	{
//	  責務: [markdownCell: Markdown tableやheadingを壊す改行・pipe文字を無害化する]
//	  処理: [1: CR/LFをspaceへ置き換える; 2: pipeをescapeする]
//	  引数: [value: Markdownへ挿入する未加工文字列]
//	  戻り値: [string: 1行内へ安全に表示できる文字列]
//	}
func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.ReplaceAll(value, "|", "\\|")
}
