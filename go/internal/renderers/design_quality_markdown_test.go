package renderers

import (
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

func TestRenderDesignQualityMarkdownIncludesEvidenceAndFinalNewline(t *testing.T) {
	actual := RenderDesignQualityMarkdown(models.DesignQualityReport{
		Metrics: []models.DesignMetric{{Name: "node_count", Value: 2, Scope: "call_graph"}},
		Findings: []models.DesignFinding{{
			Code: "call-cycle", Severity: "warning", Subject: "alpha|beta",
			Evidence: "2本のedge\nを持つcycle", Interpretation: "呼び出しgraphに依存cycleがあります。", Note: "意図を確認してください。",
		}},
	})
	expected := "# 設計品質レポート\n\n" +
		"## 指標\n\n| 対象 | 指標 | 値 |\n| --- | --- | ---: |\n" +
		"| call_graph | node_count | 2 |\n\n" +
		"## 指摘\n\n" +
		"### [WARNING] call-cycle: alpha\\|beta\n\n" +
		"- 根拠: 2本のedge を持つcycle\n" +
		"- 解釈: 呼び出しgraphに依存cycleがあります。\n" +
		"- 補足: 意図を確認してください。\n"
	if actual != expected {
		t.Fatalf("rendered report differs from golden\nactual:   %q\nexpected: %q", actual, expected)
	}
}

func TestRenderDesignQualityMarkdownExplainsEmptyFindings(t *testing.T) {
	actual := RenderDesignQualityMarkdown(models.DesignQualityReport{})
	expected := "# 設計品質レポート\n\n" +
		"## 指標\n\n| 対象 | 指標 | 値 |\n| --- | --- | ---: |\n\n" +
		"## 指摘\n\n現在の閾値では構造上の指摘は見つかりませんでした。\n"
	if actual != expected {
		t.Fatalf("empty report differs from golden\nactual:   %q\nexpected: %q", actual, expected)
	}
}
