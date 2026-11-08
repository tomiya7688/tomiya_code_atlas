package generators

import (
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [CIWorkflowDiagram: YAML構文から切り離したCI graphのtrigger・job・edgeを保持する]
//	  フィールド: [Triggers: 起動条件, Jobs: 表示するjob node, Edges: job依存relation]
//	}
type CIWorkflowDiagram struct {
	Triggers []string
	Jobs     []CIJobNode
	Edges    []CIJobEdge
}

//	{
//	  責務: [CIJobNode: CI graph上で表示するjob keyとlabelを保持する]
//	  フィールド: [ID: workflow内のjob key, Label: job名とstep名を結合した表示文]
//	}
type CIJobNode struct {
	ID    string
	Label string
}

//	{
//	  責務: [CIJobEdge: workflow triggerまたは先行jobからjobへの依存を保持する]
//	  フィールド: [Source: 依存元job ID, Target: 依存先job ID, FromTrigger: workflow triggerから始まるedgeならtrue]
//	}
type CIJobEdge struct {
	Source      string
	Target      string
	FromTrigger bool
}

//	{
//	  責務: [BuildCIWorkflowDiagram: CI workflowのjobsとneedsからrenderer非依存のnode・edgeを作る]
//	  処理: [1: triggerをfallback付きで選ぶ, 2: job labelと既知needs edgeを作る, 3: 依存のないjobをtriggerへ接続する]
//	  引数: [workflow: YAML解析済みのCI workflow model]
//	  戻り値: [CIWorkflowDiagram: 表示用nodeとedge]
//	}
func BuildCIWorkflowDiagram(workflow models.CIWorkflow) CIWorkflowDiagram {
	triggers := make([]string, 0, len(workflow.Triggers))
	for _, trigger := range workflow.Triggers {
		trimmedTrigger := strings.TrimSpace(trigger)
		if trimmedTrigger != "" {
			triggers = append(triggers, trimmedTrigger)
		}
	}
	if len(triggers) == 0 {
		triggers = []string{"workflow"}
	}
	result := CIWorkflowDiagram{Triggers: triggers, Jobs: make([]CIJobNode, 0, len(workflow.Jobs)), Edges: []CIJobEdge{}}
	knownJobs := make(map[string]struct{}, len(workflow.Jobs))
	for _, job := range workflow.Jobs {
		knownJobs[job.ID] = struct{}{}
		label := job.Name
		for _, step := range job.Steps {
			label += "\n" + step
		}
		result.Jobs = append(result.Jobs, CIJobNode{ID: job.ID, Label: label})
	}
	for _, job := range workflow.Jobs {
		addedDependency := false
		seenDependencies := make(map[string]struct{}, len(job.Needs))
		for _, dependency := range job.Needs {
			if _, exists := knownJobs[dependency]; exists {
				if _, duplicate := seenDependencies[dependency]; duplicate {
					continue
				}
				seenDependencies[dependency] = struct{}{}
				result.Edges = append(result.Edges, CIJobEdge{Source: dependency, Target: job.ID})
				addedDependency = true
			}
		}
		if !addedDependency {
			result.Edges = append(result.Edges, CIJobEdge{Target: job.ID, FromTrigger: true})
		}
	}
	return result
}
