// Package staticdoc generates renderer-neutral source reference Markdown from Common IR.
package staticdoc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

type entity struct {
	commonir.Entity
	qualified string
	resolved  []string
	calledBy  []string
}

// Render produces a deterministic static reference document from a parsed module.
func Render(module commonir.Module, sourceName string) string {
	items := make([]entity, len(module.Entities))
	callables := map[string][]int{}
	for i, item := range module.Entities {
		q := item.Name
		if item.Parent != nil && *item.Parent != "" {
			q = *item.Parent + "." + item.Name
		}
		items[i] = entity{Entity: item, qualified: q}
		if item.Kind == "function" || item.Kind == "method" {
			callables[item.Name] = appendUniqueIndex(callables[item.Name], i)
			callables[q] = appendUniqueIndex(callables[q], i)
		}
	}
	for i := range items {
		for _, call := range items[i].CallSequence {
			name := call
			if dot := strings.LastIndex(name, "."); dot >= 0 {
				name = name[dot+1:]
			}
			candidates := callables[name]
			if len(candidates) == 1 {
				target := items[candidates[0]].qualified
				items[i].resolved = appendUnique(items[i].resolved, target)
				items[candidates[0]].calledBy = appendUnique(items[candidates[0]].calledBy, items[i].qualified)
			}
			if items[i].Parent != nil && len(candidates) > 1 {
				for _, c := range candidates {
					if items[c].Parent != nil && *items[c].Parent == *items[i].Parent {
						target := items[c].qualified
						items[i].resolved = appendUnique(items[i].resolved, target)
						items[c].calledBy = appendUnique(items[c].calledBy, items[i].qualified)
					}
				}
			}
		}
	}
	for i := range items {
		sort.Strings(items[i].calledBy)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n- 言語: %s\n", sourceName, module.Language)
	if module.ModuleDocstring != nil && strings.TrimSpace(*module.ModuleDocstring) != "" {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(*module.ModuleDocstring))
	}
	if len(items) == 0 {
		b.WriteString("\n宣言されたクラス、関数、メソッドはありません。\n")
		return b.String()
	}
	b.WriteString("\n## 宣言一覧\n")
	for _, item := range items {
		level := 2
		if item.Kind == "method" {
			level = 3
		}
		heading := strings.Repeat("#", level)
		kind := item.Kind
		if item.DeclarationKind != nil && *item.DeclarationKind != "" {
			kind = *item.DeclarationKind
		}
		fmt.Fprintf(&b, "\n%s %s: `%s`\n\n", heading, kind, item.qualified)
		location := fmt.Sprintf("- 定義位置: %d行目", item.Line)
		if item.EndLine != item.Line {
			location += fmt.Sprintf("（%d行目まで）", item.EndLine)
		}
		b.WriteString(location + "\n")
		fmt.Fprintf(&b, "- 公開範囲: %s\n", item.Visibility)
		if len(item.Bases) > 0 {
			b.WriteString("- 継承・実装: ")
			b.WriteString(codeList(item.Bases))
			b.WriteByte('\n')
		}
		if signature := signature(item); signature != "" {
			fmt.Fprintf(&b, "- 宣言: `%s`\n", signature)
		}
		if item.Docstring != nil && strings.TrimSpace(*item.Docstring) != "" {
			fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(*item.Docstring))
		}
		if len(item.Calls) > 0 {
			fmt.Fprintf(&b, "- 呼び出し式: %s\n", codeList(item.Calls))
		}
		if len(item.resolved) > 0 {
			fmt.Fprintf(&b, "- 同一ファイル内で解決した呼び出し先: %s\n", codeList(item.resolved))
		}
		if len(item.calledBy) > 0 {
			fmt.Fprintf(&b, "- 呼び出し元: %s\n", codeList(item.calledBy))
		}
	}
	return b.String()
}

func signature(item entity) string {
	if item.Kind != "function" && item.Kind != "method" {
		return ""
	}
	types := map[string]string{}
	for _, pair := range item.ParameterTypes {
		if len(pair) >= 2 {
			types[pair[0]] = pair[1]
		}
	}
	params := append([]string{}, item.Parameters...)
	if item.Kind == "method" && len(params) > 0 && (params[0] == "self" || params[0] == "cls") {
		params = params[1:]
	}
	for i, p := range params {
		if t := types[strings.TrimLeft(p, "*")]; t != "" {
			params[i] += ": " + t
		}
	}
	result := item.Name + "(" + strings.Join(params, ", ") + ")"
	if item.ReturnType != nil {
		result += " -> " + *item.ReturnType
	}
	return result
}

func codeList(values []string) string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = "`" + strings.ReplaceAll(value, "`", "\\`") + "`"
	}
	return strings.Join(out, ", ")
}
func appendUnique(values []string, value string) []string {
	for _, old := range values {
		if old == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueIndex(values []int, value int) []int {
	for _, old := range values {
		if old == value {
			return values
		}
	}
	return append(values, value)
}
