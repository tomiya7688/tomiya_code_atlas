package models

//	{
//	  責務: [DesignMetric: 設計評価で計測した名前・値・対象範囲を保持する]
//	  フィールド: [Name: metric識別子; Value: 数値または表示値; Scope: graphなどの適用範囲]
//	}
type DesignMetric struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
	Scope string `json:"scope"`
}

//	{
//	  責務: [DesignFinding: 解析根拠に基づく設計上の指摘を保持する]
//	  フィールド: [Code: 指摘種別; Severity: 重要度; Subject: 対象; Evidence: 観測事実; Interpretation: 説明; Note: 補足]
//	}
type DesignFinding struct {
	Code           string `json:"code"`
	Severity       string `json:"severity"`
	Subject        string `json:"subject"`
	Evidence       string `json:"evidence"`
	Interpretation string `json:"interpretation"`
	Note           string `json:"note,omitempty"`
}

//	{
//	  責務: [DesignQualityReport: 設計評価のmetricとfindingをrenderer非依存でまとめる]
//	  フィールド: [Metrics: 計測値; Findings: 根拠付きの指摘]
//	}
type DesignQualityReport struct {
	Metrics  []DesignMetric  `json:"metrics"`
	Findings []DesignFinding `json:"findings"`
}
