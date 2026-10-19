package evaluators

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

func TestDesignQualityMatchesSharedGolden(t *testing.T) {
	fixtureBytes, err := os.ReadFile("../../../tests/fixtures/evaluators/design_quality_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Module     commonir.Module         `json:"module"`
		Thresholds DesignQualityThresholds `json:"thresholds"`
		Expected   struct {
			FindingCodes    []string `json:"finding_codes"`
			FindingSubjects []string `json:"finding_subjects"`
			CallCycleCount  int      `json:"call_cycle_count"`
		} `json:"expected"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	report, err := EvaluateDesignQuality(fixture.Module, &fixture.Thresholds)
	if err != nil {
		t.Fatal(err)
	}
	actualCodes := make([]string, len(report.Findings))
	actualSubjects := make([]string, len(report.Findings))
	for index, finding := range report.Findings {
		actualCodes[index] = finding.Code
		actualSubjects[index] = finding.Subject
	}
	if !reflect.DeepEqual(actualCodes, fixture.Expected.FindingCodes) {
		t.Fatalf("finding codes = %v, want %v", actualCodes, fixture.Expected.FindingCodes)
	}
	if !reflect.DeepEqual(actualSubjects, fixture.Expected.FindingSubjects) {
		t.Fatalf("finding subjects = %v, want %v", actualSubjects, fixture.Expected.FindingSubjects)
	}
	if metric := findMetric(t, report.Metrics, "cycle_count", "call_graph"); metric.Value != fixture.Expected.CallCycleCount {
		t.Fatalf("call graph cycle count = %v, want %d", metric.Value, fixture.Expected.CallCycleCount)
	}
}

func TestEvaluateDesignQualityReportsCycleWithEvidence(t *testing.T) {
	report, err := EvaluateDesignQuality(commonir.Module{Entities: []commonir.Entity{
		{Kind: "function", Name: "alpha", Calls: []string{"beta"}},
		{Kind: "function", Name: "beta", Calls: []string{"alpha"}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	finding := findFinding(t, report.Findings, "call-cycle")
	if finding.Severity != "warning" || !strings.Contains(finding.Subject, "alpha") || !strings.Contains(finding.Subject, "beta") {
		t.Fatalf("unexpected cycle finding: %#v", finding)
	}
	if !strings.Contains(finding.Evidence, "edge") || !strings.Contains(finding.Interpretation, "cycle") {
		t.Fatalf("cycle finding must explain its evidence: %#v", finding)
	}
	if metric := findMetric(t, report.Metrics, "cycle_count", "call_graph"); metric.Value != 1 {
		t.Fatalf("call graph cycle count = %v, want 1", metric.Value)
	}
}

func TestHighFanInIsInformationalAndLocalized(t *testing.T) {
	module := commonir.Module{Entities: []commonir.Entity{
		{Kind: "function", Name: "first", Calls: []string{"shared"}},
		{Kind: "function", Name: "second", Calls: []string{"shared"}},
		{Kind: "function", Name: "third", Calls: []string{"shared"}},
		{Kind: "function", Name: "shared"},
	}}
	thresholds := DefaultDesignQualityThresholds()
	thresholds.HighFanOut = 10
	thresholds.LargeSeries = 10
	report, err := EvaluateDesignQuality(module, &thresholds)
	if err != nil {
		t.Fatal(err)
	}
	finding := findFinding(t, report.Findings, "high-fan-in")
	if finding.Subject != "shared" || finding.Severity != "info" || !strings.Contains(finding.Evidence, "3") {
		t.Fatalf("unexpected high fan-in finding: %#v", finding)
	}
	if !strings.Contains(finding.Interpretation, "複数箇所") || finding.Note == "" {
		t.Fatalf("finding should use Japanese and explain legitimate hubs: %#v", finding)
	}
}

func TestHighFanOutAndLargeSeriesAreWarnings(t *testing.T) {
	module := commonir.Module{Entities: []commonir.Entity{
		{Kind: "function", Name: "main", Calls: []string{"one", "two", "three"}},
		{Kind: "function", Name: "one"},
		{Kind: "function", Name: "two"},
		{Kind: "function", Name: "three"},
	}}
	thresholds := DefaultDesignQualityThresholds()
	thresholds.HighFanIn = 10
	thresholds.HighFanOut = 3
	thresholds.LargeSeries = 4
	report, err := EvaluateDesignQuality(module, &thresholds)
	if err != nil {
		t.Fatal(err)
	}
	if finding := findFinding(t, report.Findings, "high-fan-out"); finding.Subject != "main" || finding.Severity != "warning" {
		t.Fatalf("unexpected high fan-out finding: %#v", finding)
	}
	if finding := findFinding(t, report.Findings, "large-series"); finding.Severity != "warning" || !strings.Contains(finding.Evidence, "4") {
		t.Fatalf("unexpected large-series finding: %#v", finding)
	}
}

func TestClassCycleUsesCommonRelationAnalysis(t *testing.T) {
	module := commonir.Module{Entities: []commonir.Entity{
		{Kind: "class", Name: "Alpha", Bases: []string{"Beta"}},
		{Kind: "class", Name: "Beta", Bases: []string{"Alpha"}},
	}}
	thresholds := DefaultDesignQualityThresholds()
	thresholds.HighFanIn = 10
	thresholds.HighFanOut = 10
	thresholds.LargeSeries = 10
	report, err := EvaluateDesignQuality(module, &thresholds)
	if err != nil {
		t.Fatal(err)
	}
	finding := findFinding(t, report.Findings, "class-cycle")
	if !strings.Contains(finding.Subject, "Alpha") || !strings.Contains(finding.Subject, "Beta") {
		t.Fatalf("class cycle did not preserve both class names: %#v", finding)
	}
}

func TestInvalidDesignQualityThresholdsAreRejected(t *testing.T) {
	thresholds := DefaultDesignQualityThresholds()
	thresholds.HighFanIn = 0
	if _, err := EvaluateDesignQuality(commonir.Module{}, &thresholds); err == nil {
		t.Fatal("expected invalid integer threshold error")
	}
	thresholds = DefaultDesignQualityThresholds()
	thresholds.HighCrossSeriesRatio = 1.1
	if _, err := EvaluateDesignQuality(commonir.Module{}, &thresholds); err == nil {
		t.Fatal("expected invalid ratio threshold error")
	}
	thresholds.HighCrossSeriesRatio = math.NaN()
	if _, err := EvaluateDesignQuality(commonir.Module{}, &thresholds); err == nil {
		t.Fatal("expected NaN ratio threshold error")
	}
}

func findFinding(t *testing.T, findings []models.DesignFinding, code string) models.DesignFinding {
	t.Helper()
	for _, finding := range findings {
		if finding.Code == code {
			return finding
		}
	}
	t.Fatalf("finding %q not found in %#v", code, findings)
	return models.DesignFinding{}
}

func findMetric(t *testing.T, metrics []models.DesignMetric, name, scope string) models.DesignMetric {
	t.Helper()
	for _, metric := range metrics {
		if metric.Name == name && metric.Scope == scope {
			return metric
		}
	}
	t.Fatalf("metric %q in %q not found in %#v", name, scope, metrics)
	return models.DesignMetric{}
}
