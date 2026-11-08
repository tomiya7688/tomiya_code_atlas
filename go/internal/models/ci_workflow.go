package models

//	{
//	  責務: [CIWorkflow: GitHub Actions workflowのtriggerとjobを言語非依存の図用dataとして保持する]
//	  フィールド: [Name: workflow名, Triggers: 起動条件, Jobs: 表示するjob一覧]
//	}
type CIWorkflow struct {
	Name     string
	Triggers []string
	Jobs     []CIJob
}

//	{
//	  責務: [CIJob: jobの識別子・依存関係・手順名を保持する]
//	  フィールド: [ID: YAML上のjob key, Name: 表示名, Needs: 先行job key, Steps: 図に表示する手順名]
//	}
type CIJob struct {
	ID    string
	Name  string
	Needs []string
	Steps []string
}
