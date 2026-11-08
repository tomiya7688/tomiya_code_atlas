package analyzers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [ParseGitHubActionsWorkflow: GitHub Actions YAMLからtrigger・job・needs・step名だけを正規化する]
//	  処理: [1: YAML documentをmappingとして読む, 2: jobsと依存を検証して順序を固定する, 3: 実行内容を評価せずCI図modelを返す]
//	  引数: [source: workflow YAML本文, sourceName: errorに表示するfile名]
//	  戻り値: [workflow: 解析したCI graph model, error: YAML rootまたはjobsが不正な場合の説明]
//	}
func ParseGitHubActionsWorkflow(source, sourceName string) (models.CIWorkflow, error) {
	documents, err := decodeYAMLDocuments(source, sourceName)
	if err != nil {
		return models.CIWorkflow{}, err
	}
	if len(documents) == 0 {
		return models.CIWorkflow{}, fmt.Errorf("GitHub Actions workflowが空です (%s)", sourceName)
	}
	document := documents[0]
	rawJobs, exists := document["jobs"]
	if !exists || rawJobs == nil {
		return models.CIWorkflow{}, fmt.Errorf("GitHub Actions jobsが見つかりません (%s)", sourceName)
	}
	jobMap, ok := rawJobs.(map[string]any)
	if !ok {
		return models.CIWorkflow{}, fmt.Errorf("GitHub Actions jobsはmappingである必要があります (%s)", sourceName)
	}

	workflow := models.CIWorkflow{Name: ciString(document["name"])}
	workflow.Triggers = normalizeCITriggers(document)
	jobIDs := make([]string, 0, len(jobMap))
	for jobID := range jobMap {
		jobIDs = append(jobIDs, jobID)
	}
	sort.Strings(jobIDs)
	for _, jobID := range jobIDs {
		jobConfig, ok := jobMap[jobID].(map[string]any)
		if !ok {
			continue
		}
		job := models.CIJob{ID: jobID, Name: ciString(jobConfig["name"])}
		if job.Name == "" {
			job.Name = jobID
		}
		job.Needs = normalizeCINeeds(jobConfig["needs"])
		job.Steps = normalizeCISteps(jobConfig["steps"])
		workflow.Jobs = append(workflow.Jobs, job)
	}
	if len(workflow.Jobs) == 0 {
		return models.CIWorkflow{}, fmt.Errorf("GitHub Actions workflowに有効なjobがありません (%s)", sourceName)
	}
	return workflow, nil
}

//	{
//	  責務: [normalizeCITriggers: workflowのon値からtrigger名を安定した順序で抽出する]
//	  引数: [document: decode済みworkflow mapping]
//	  戻り値: [[]string: trigger名。未指定なら空slice]
//	}
func normalizeCITriggers(document map[string]any) []string {
	value := document["on"]
	triggers := []string{}
	switch typed := value.(type) {
	case string:
		triggers = append(triggers, typed)
	case []any:
		for _, item := range typed {
			if trigger, ok := item.(string); ok {
				triggers = append(triggers, trigger)
			}
		}
	case map[string]any:
		for trigger := range typed {
			triggers = append(triggers, trigger)
		}
	}
	sort.Strings(triggers)
	return triggers
}

//	{
//	  責務: [normalizeCINeeds: job needsを一意・安定した依存ID一覧へ変換する]
//	  引数: [value: YAMLから読み取ったneeds値]
//	  戻り値: [[]string: 依存job ID]
//	}
func normalizeCINeeds(value any) []string {
	needs := []string{}
	switch typed := value.(type) {
	case string:
		needs = append(needs, typed)
	case []any:
		for _, item := range typed {
			if need, ok := item.(string); ok {
				needs = append(needs, need)
			}
		}
	}
	sort.Strings(needs)
	return needs
}

//	{
//	  責務: [normalizeCISteps: job stepsを利用者が図で読める短いlabelへ変換する]
//	  引数: [value: YAMLから読み取ったsteps値]
//	  戻り値: [[]string: stepの表示label]
//	}
func normalizeCISteps(value any) []string {
	rawSteps, ok := value.([]any)
	if !ok {
		return []string{}
	}
	steps := make([]string, 0, len(rawSteps))
	for index, rawStep := range rawSteps {
		step, ok := rawStep.(map[string]any)
		if !ok {
			continue
		}
		label := ciString(step["name"])
		if label == "" {
			label = ciString(step["uses"])
		}
		if label == "" {
			command := strings.TrimSpace(ciString(step["run"]))
			label, _, _ = strings.Cut(command, "\n")
		}
		if label == "" {
			label = fmt.Sprintf("step %d", index+1)
		}
		steps = append(steps, label)
	}
	return steps
}

//	{
//	  責務: [ciString: YAML値から表示可能なstringだけを安全に取り出す]
//	  引数: [value: YAML decoderが返した任意値]
//	  戻り値: [string: string型の値。その他は空文字]
//	}
func ciString(value any) string {
	text, _ := value.(string)
	return text
}
