package renderers

import (
	"encoding/csv"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

//	{
//	  責務: [RenderResponsibilityMarkdown: class責務rowsを安定したMarkdown tableへ直列化する]
//	}
func RenderResponsibilityMarkdown(rows []generators.ResponsibilityRow) string {
	lines := []string{"| Class | Responsibility |", "| --- | --- |"}
	for _, row := range rows {
		lines = append(lines, "| "+row.ClassName+" | "+row.Responsibility+" |")
	}
	return strings.Join(lines, "\n") + "\n"
}

//	{
//	  責務: [RenderResponsibilityCSV: class責務rowsをRFC4180互換のCSVへ直列化する]
//	}
func RenderResponsibilityCSV(rows []generators.ResponsibilityRow) string {
	var output strings.Builder
	writer := csv.NewWriter(&output)
	writer.UseCRLF = false
	_ = writer.Write([]string{"Class", "Responsibility"})
	for _, row := range rows {
		_ = writer.Write([]string{row.ClassName, row.Responsibility})
	}
	writer.Flush()
	return output.String()
}
