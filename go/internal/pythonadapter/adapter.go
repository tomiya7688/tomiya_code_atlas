// Package pythonadapter extracts the declaration subset needed by shared
// generators without exposing parser-specific syntax outside this boundary.
package pythonadapter

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

var declaration = regexp.MustCompile(`^(async\s+)?(def|class)\s+([A-Za-z_][A-Za-z0-9_]*)`)
var identifier = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// Parse converts Python source into the language-neutral Common IR v1 model.
// It intentionally extracts stable source facts only; it never executes code.
func Parse(source string) commonir.Payload {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	module := commonir.Module{Language: "python", Entities: []commonir.Entity{}, Imports: []string{}, Diagnostics: []commonir.Diagnostic{}}
	if doc, _, ok := docstringAt(lines, firstCodeLine(lines)); ok {
		module.ModuleDocstring = &doc
	}
	type scope struct {
		entity   int
		indent   int
		kind     string
		awaitDoc bool
	}
	var stack []scope
	var decorators []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := indentation(line)
		for len(stack) > 0 && indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}
		for _, active := range stack {
			module.Entities[active.entity].EndLine = i + 1
		}
		if strings.HasPrefix(trimmed, "@") {
			decorators = append(decorators, strings.TrimSpace(strings.TrimPrefix(trimmed, "@")))
			continue
		}
		logical := strings.TrimSpace(stripComment(line))
		decl := declaration.FindStringSubmatch(logical)
		if decl != nil {
			name, kind := decl[3], decl[2]
			start := i + 1
			full := logical
			for parenBalance(full) > 0 && i+1 < len(lines) {
				i++
				full += " " + strings.TrimSpace(stripComment(lines[i]))
			}
			parent := (*string)(nil)
			if len(stack) > 0 {
				p := qualified(module.Entities, stack[len(stack)-1].entity)
				parent = &p
			}
			entityKind, visibility := "function", "public"
			if kind == "class" {
				entityKind = "class"
			} else if parent != nil && stack[len(stack)-1].kind == "class" {
				entityKind = "method"
			}
			if strings.HasPrefix(name, "__") && !strings.HasSuffix(name, "__") {
				visibility = "private"
			} else if strings.HasPrefix(name, "_") && !(strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")) {
				visibility = "protected"
			}
			entity := commonir.Entity{Kind: entityKind, Name: name, SourceLocation: commonir.SourceLocation{Line: start, EndLine: start}, Indent: indent, Parent: parent, Parameters: []string{}, Decorators: append([]string{}, decorators...), Calls: []string{}, CallSequence: []string{}, Visibility: visibility, Bases: []string{}, TypeParameters: []string{}, TypeConstraints: []string{}, ParameterTypes: [][]string{}}
			decorators = nil
			if decl[1] != "" {
				entity.IsAsync = true
			}
			if kind == "class" {
				entity.Bases = classBases(full)
			} else {
				parseSignature(full, &entity)
			}
			module.Entities = append(module.Entities, entity)
			stack = append(stack, scope{entity: len(module.Entities) - 1, indent: indent, kind: kind, awaitDoc: true})
			continue
		}
		decorators = nil
		if len(stack) > 0 && stack[len(stack)-1].awaitDoc {
			if doc, next, ok := docstringAt(lines, i); ok {
				module.Entities[stack[len(stack)-1].entity].Docstring = &doc
				stack[len(stack)-1].awaitDoc = false
				for _, active := range stack {
					module.Entities[active.entity].EndLine = next
				}
				i = next - 1
				continue
			}
			stack[len(stack)-1].awaitDoc = false
		}
		if len(stack) > 0 {
			current := &module.Entities[stack[len(stack)-1].entity]
			current.EndLine = i + 1
			calls := callsInLine(logical)
			current.Calls = append(current.Calls, calls...)
			current.CallSequence = append(current.CallSequence, calls...)
		}
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			module.Imports = append(module.Imports, strings.TrimSpace(logical))
		}
	}
	for i := range module.Entities {
		module.Entities[i].Calls = unique(module.Entities[i].Calls)
	}
	resolveCalls(module.Entities)
	return commonir.Payload{SchemaVersion: commonir.SchemaVersion, Module: module}
}

func resolveCalls(entities []commonir.Entity) {
	byName := map[string][]int{}
	for i, entity := range entities {
		if entity.Kind == "function" || entity.Kind == "method" {
			byName[entity.Name] = append(byName[entity.Name], i)
		}
	}
	for i := range entities {
		for _, call := range entities[i].Calls {
			leaf := call
			if dot := strings.LastIndex(leaf, "."); dot >= 0 {
				leaf = leaf[dot+1:]
			}
			candidates := byName[leaf]
			if len(candidates) == 1 {
				entities[i].ResolvedCalls = append(entities[i].ResolvedCalls, qualified(entities, candidates[0]))
				continue
			}
			if entities[i].Parent != nil {
				for _, target := range candidates {
					if entities[target].Parent != nil && qualified(entities, i) == *entities[target].Parent {
						entities[i].ResolvedCalls = append(entities[i].ResolvedCalls, qualified(entities, target))
					}
				}
			}
		}
		entities[i].ResolvedCalls = unique(entities[i].ResolvedCalls)
	}
}

func indentation(line string) int {
	n := 0
	for _, r := range line {
		if r == ' ' {
			n++
		} else if r == '\t' {
			n += 8 - n%8
		} else {
			break
		}
	}
	return n
}

func stripComment(s string) string {
	var quote rune
	triple := false
	escaped := false
	for i, r := range s {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				if triple && strings.HasPrefix(s[i:], strings.Repeat(string(quote), 3)) {
					quote = 0
					triple = false
				}
				if !triple {
					quote = 0
				}
			}
			continue
		}
		if r == '#' {
			return s[:i]
		}
		if r == '\'' || r == '"' {
			quote = r
			triple = strings.HasPrefix(s[i:], strings.Repeat(string(r), 3))
		}
	}
	return s
}

func parenBalance(s string) int {
	var quote rune
	escaped, depth := false, 0
	for _, r := range s {
		if quote != 0 {
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == '(' || r == '[' || r == '{' {
			depth++
		}
		if r == ')' || r == ']' || r == '}' {
			depth--
		}
	}
	return depth
}

func classBases(s string) []string {
	start := strings.Index(s, "class ")
	if start < 0 {
		return []string{}
	}
	open := strings.Index(s[start:], "(")
	if open < 0 {
		return []string{}
	}
	open += start
	end := strings.Index(s[open:], ")")
	if end < 0 {
		return []string{}
	}
	end += open
	var out []string
	for _, item := range splitTopLevel(s[open+1:end], ',') {
		value := strings.TrimSpace(item)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func parseSignature(s string, entity *commonir.Entity) {
	open := strings.Index(s, "(")
	if open < 0 {
		return
	}
	close := matchingParen(s, open)
	if close < 0 {
		return
	}
	for _, item := range splitTopLevel(s[open+1:close], ',') {
		item = strings.TrimSpace(item)
		if item == "" || item == "/" || item == "*" {
			continue
		}
		if eq := strings.Index(item, "="); eq >= 0 {
			item = strings.TrimSpace(item[:eq])
		}
		name, annotation := item, ""
		if colon := strings.Index(item, ":"); colon >= 0 {
			name, annotation = strings.TrimSpace(item[:colon]), strings.TrimSpace(item[colon+1:])
		}
		parameter := name
		name = strings.TrimLeft(name, "*")
		if name != "" {
			entity.Parameters = append(entity.Parameters, parameter)
			if annotation != "" {
				entity.ParameterTypes = append(entity.ParameterTypes, []string{name, annotation})
			}
		}
	}
	if arrow := strings.Index(s[close+1:], "->"); arrow >= 0 {
		result := strings.TrimSpace(s[close+1+arrow+2:])
		result = strings.TrimSuffix(result, ":")
		if result != "" {
			entity.ReturnType = &result
		}
	}
}

func matchingParen(s string, open int) int {
	depth := 0
	var quote rune
	escaped := false
	for i, r := range s[open:] {
		index := open + i
		if quote != 0 {
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == '(' {
			depth++
		}
		if r == ')' {
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func splitTopLevel(s string, delimiter rune) []string {
	var out []string
	start, depth := 0, 0
	var quote rune
	escaped := false
	for i, r := range s {
		if quote != 0 {
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == '(' || r == '[' || r == '{' {
			depth++
		}
		if r == ')' || r == ']' || r == '}' {
			depth--
		}
		if r == delimiter && depth == 0 {
			out = append(out, s[start:i])
			start = i + len(string(r))
		}
	}
	return append(out, s[start:])
}

func firstCodeLine(lines []string) int {
	for i, line := range lines {
		s := strings.TrimSpace(line)
		if s != "" && !strings.HasPrefix(s, "#") {
			return i
		}
	}
	return -1
}

func docstringAt(lines []string, start int) (string, int, bool) {
	if start < 0 || start >= len(lines) {
		return "", 0, false
	}
	s := strings.TrimSpace(lines[start])
	for _, prefix := range []string{"r", "u", "f", "fr", "rf", "b", "br", "rb"} {
		if strings.HasPrefix(strings.ToLower(s), prefix+"\"\"\"") || strings.HasPrefix(strings.ToLower(s), prefix+"'''") {
			s = s[len(prefix):]
			break
		}
	}
	for _, quote := range []string{"\"\"\"", "'''"} {
		if !strings.HasPrefix(s, quote) {
			continue
		}
		body := s[len(quote):]
		if end := strings.Index(body, quote); end >= 0 {
			return strings.TrimSpace(body[:end]), start + 1, true
		}
		parts := []string{body}
		for i := start + 1; i < len(lines); i++ {
			if end := strings.Index(lines[i], quote); end >= 0 {
				parts = append(parts, lines[i][:end])
				return strings.TrimSpace(strings.Join(parts, "\n")), i + 1, true
			}
			parts = append(parts, lines[i])
		}
		return "", start + 1, false
	}
	for _, quote := range []string{"\"", "'"} {
		if strings.HasPrefix(s, quote) && strings.HasSuffix(s, quote) && len(s) >= 2 {
			return strings.TrimSpace(s[1 : len(s)-1]), start + 1, true
		}
	}
	return "", start + 1, false
}

func docstringLiteral(s string) (string, bool) {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"r", "u", "f", "fr", "rf", "b", "br", "rb"} {
		if strings.HasPrefix(strings.ToLower(s), prefix+"\"\"\"") || strings.HasPrefix(strings.ToLower(s), prefix+"'''") {
			s = s[len(prefix):]
			break
		}
	}
	for _, q := range []string{"\"\"\"", "'''", "\"", "'"} {
		if strings.HasPrefix(s, q) && strings.HasSuffix(s, q) && len(s) >= 2*len(q) {
			return strings.TrimSpace(s[len(q) : len(s)-len(q)]), true
		}
	}
	return "", false
}

func callsInLine(s string) []string {
	var calls []string
	for _, loc := range identifier.FindAllStringIndex(s, -1) {
		name := s[loc[0]:loc[1]]
		if isKeyword(name) {
			continue
		}
		before := strings.TrimSpace(s[:loc[0]])
		if strings.HasSuffix(before, "def") || strings.HasSuffix(before, "class") || strings.HasSuffix(before, "@") {
			continue
		}
		j := loc[1]
		for j < len(s) && unicode.IsSpace(rune(s[j])) {
			j++
		}
		if j < len(s) && s[j] == '(' {
			start := loc[0]
			for start > 0 && (s[start-1] == '.' || isIdentifierByte(s[start-1])) {
				start--
			}
			value := strings.TrimSpace(s[start:loc[1]])
			if value != "" {
				calls = append(calls, value)
			}
		}
	}
	return calls
}

func isIdentifierByte(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}
func isKeyword(s string) bool {
	switch s {
	case "if", "for", "while", "with", "return", "yield", "assert", "raise", "def", "class", "lambda", "and", "or", "not", "in", "is", "async", "await", "match", "case", "print":
		return true
	}
	return false
}
func unique(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
func qualified(entities []commonir.Entity, index int) string {
	e := entities[index]
	if e.Parent == nil || *e.Parent == "" {
		return e.Name
	}
	return *e.Parent + "." + e.Name
}

// Validate rejects unsupported protocol versions at the adapter boundary.
func Validate(payload commonir.Payload) error {
	if payload.SchemaVersion != commonir.SchemaVersion || payload.Language != "python" {
		return fmt.Errorf("invalid Python adapter payload")
	}
	return nil
}
