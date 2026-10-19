package renderers

import (
	"fmt"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [RenderStaticSpecificationMarkdown: 静的仕様書modelを日本語Markdownへ変換する]
//	  引数: [document: 静的仕様書の論理model]
//	  戻り値: [string: 改行で終わるMarkdown]
//	}
func RenderStaticSpecificationMarkdown(document models.StaticSpecification) string {
	lines := []string{"# " + document.SourceName, "", "- 言語: " + document.Language}
	if document.ModuleDocstring != "" {
		lines = append(lines, "", strings.TrimSpace(document.ModuleDocstring))
	}
	if len(document.Declarations) == 0 {
		lines = append(lines, "", "宣言されたクラス、関数、メソッドはありません。")
		return strings.Join(lines, "\n") + "\n"
	}
	lines = append(lines, "", "## 宣言一覧")
	for _, declaration := range document.Declarations {
		lines = append(lines, "")
		lines = append(lines, renderStaticDeclaration(declaration)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

//	{
//	  責務: [renderStaticDeclaration: 宣言model一件のMarkdown行を組み立てる]
//	  引数: [declaration: 静的宣言model]
//	  戻り値: [[]string: Markdown行]
//	}
func renderStaticDeclaration(declaration models.StaticDeclaration) []string {
	headingLevel := 2
	if declaration.Kind == "method" {
		headingLevel = 3
	}
	kind := declaration.DeclarationKind
	if kind == "" {
		kind = declaration.Kind
	}
	lines := []string{strings.Repeat("#", headingLevel) + fmt.Sprintf(" %s: `%s`", kind, declaration.QualifiedName), ""}
	location := fmt.Sprintf("- 定義位置: %d行目", declaration.Line)
	if declaration.EndLine != declaration.Line {
		location += fmt.Sprintf("（%d行目まで）", declaration.EndLine)
	}
	lines = append(lines, location, "- 公開範囲: "+declaration.Visibility)
	if len(declaration.Bases) > 0 {
		lines = append(lines, "- 継承・実装: "+joinCode(declaration.Bases))
	}
	if signature := staticDeclarationSignature(declaration); signature != "" {
		lines = append(lines, "- 宣言: `"+escapeCode(signature)+"`")
	}
	if declaration.Docstring != "" {
		lines = append(lines, "", strings.TrimSpace(declaration.Docstring))
	}
	if len(declaration.Calls) > 0 {
		lines = append(lines, "- 呼び出し式: "+joinCode(declaration.Calls))
	}
	if len(declaration.ResolvedCalls) > 0 {
		lines = append(lines, "- 同一ファイル内で解決した呼び出し先: "+joinCode(declaration.ResolvedCalls))
	}
	if len(declaration.CalledBy) > 0 {
		lines = append(lines, "- 呼び出し元: "+joinCode(declaration.CalledBy))
	}
	return lines
}

//	{
//	  責務: [staticDeclarationSignature: 関数・メソッド・generic classの宣言文字列を作る]
//	  引数: [declaration: 静的宣言model]
//	  戻り値: [string: 型注釈を含む宣言文字列]
//	}
func staticDeclarationSignature(declaration models.StaticDeclaration) string {
	if declaration.Kind != "function" && declaration.Kind != "method" {
		if declaration.Kind == "class" && len(declaration.TypeParameters) > 0 {
			return declaration.Name + "[" + strings.Join(declaration.TypeParameters, ", ") + "]"
		}
		return ""
	}
	annotations := make(map[string]string, len(declaration.ParameterTypes))
	for _, pair := range declaration.ParameterTypes {
		if len(pair) >= 2 {
			annotations[pair[0]] = pair[1]
		}
	}
	parameters := declaration.Parameters
	if declaration.Kind == "method" && len(parameters) > 0 && (parameters[0] == "self" || parameters[0] == "cls") {
		parameters = parameters[1:]
	}
	formatted := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		name := strings.TrimLeft(parameter, "*")
		if annotation := annotations[name]; annotation != "" {
			formatted = append(formatted, parameter+": "+annotation)
		} else {
			formatted = append(formatted, parameter)
		}
	}
	result := declaration.Name + "(" + strings.Join(formatted, ", ") + ")"
	if declaration.ReturnType != "" {
		result += " -> " + declaration.ReturnType
	}
	return result
}

//	{
//	  責務: [joinCode: code spanで表示する複数値をMarkdown形式で連結する]
//	  引数: [values: code表記する値]
//	  戻り値: [string: code span連結文字列]
//	}
func joinCode(values []string) string {
	formatted := make([]string, 0, len(values))
	for _, value := range values {
		formatted = append(formatted, "`"+escapeCode(value)+"`")
	}
	return strings.Join(formatted, ", ")
}

//	{
//	  責務: [escapeCode: code span内のbacktickをescapeする]
//	  引数: [value: 表示する文字列]
//	  戻り値: [string: Markdown code span向け文字列]
//	}
func escapeCode(value string) string {
	return strings.ReplaceAll(value, "`", "\\`")
}
