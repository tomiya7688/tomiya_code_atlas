package analyzers

import (
	"reflect"
	"testing"
)

//	{
//	  責務: [TestParseGitHubActionsWorkflowNormalizesGraphFacts: trigger・job依存・step表示名を安定して抽出する]
//	}
func TestParseGitHubActionsWorkflowNormalizesGraphFacts(t *testing.T) {
	source := "name: Build and Test\non: [push, pull_request]\njobs:\n  test:\n    needs: build\n    steps:\n      - name: Run tests\n        run: go test ./...\n  build:\n    steps:\n      - uses: actions/checkout@v4\n"
	workflow, err := ParseGitHubActionsWorkflow(source, ".github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("ParseGitHubActionsWorkflow() error = %v", err)
	}
	if workflow.Name != "Build and Test" || !reflect.DeepEqual(workflow.Triggers, []string{"pull_request", "push"}) {
		t.Fatalf("workflow = %#v", workflow)
	}
	if len(workflow.Jobs) != 2 || workflow.Jobs[0].ID != "build" || workflow.Jobs[1].ID != "test" {
		t.Fatalf("jobs = %#v", workflow.Jobs)
	}
	if !reflect.DeepEqual(workflow.Jobs[0].Steps, []string{"actions/checkout@v4"}) || !reflect.DeepEqual(workflow.Jobs[1].Needs, []string{"build"}) || !reflect.DeepEqual(workflow.Jobs[1].Steps, []string{"Run tests"}) {
		t.Fatalf("job facts = %#v", workflow.Jobs)
	}
}

//	{
//	  責務: [TestParseGitHubActionsWorkflowRejectsInvalidJobs: jobsがmappingでないworkflowを図として受理しない]
//	}
func TestParseGitHubActionsWorkflowRejectsInvalidJobs(t *testing.T) {
	if _, err := ParseGitHubActionsWorkflow("name: Invalid\njobs: []\n", "invalid.yml"); err == nil {
		t.Fatal("ParseGitHubActionsWorkflow() accepted a non-mapping jobs value")
	}
}
