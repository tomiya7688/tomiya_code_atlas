package generators

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

var safeUseCaseChars = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

type UseCase struct {
	Name, Actor, Component, Event, Handler string
	CallChain                              []string
}

type UseCaseDiagram struct {
	Name, Scope string
	UseCases    []UseCase
}

type UseCaseDiagramBundle struct {
	Diagrams   []UseCaseDiagram
	Statistics map[string]int
}

//	{
//	  責務: [BuildUseCaseDiagramBundle: explicit input eventをhandler call chain付きuse caseへ変換する]
//	  引数: [module: Common IR, inputEvents: nilならmoduleのevents, maxDepth: internal call chain最大深さ]
//	}
func BuildUseCaseDiagramBundle(module commonir.Module, inputEvents []commonir.InputEvent, maxDepth int) (UseCaseDiagramBundle, error) {
	if maxDepth < 0 {
		return UseCaseDiagramBundle{}, fmt.Errorf("max depth must be >= 0")
	}
	events := inputEvents
	if events == nil {
		events = module.InputEvents
	}
	sequences := analyzers.ResolveCallSequences(module)
	grouped := map[string][]UseCase{}
	unresolved := 0
	for _, event := range events {
		handler := analyzers.ResolveCallableReference(module, event.Handler, event.Scope)
		chain := []string{}
		if _, exists := sequences[handler]; exists {
			visited := map[string]struct{}{handler: {}}
			var walk func(string, int)
			walk = func(node string, depth int) {
				if depth >= maxDepth {
					return
				}
				for _, call := range sequences[node] {
					if _, internal := sequences[call.Target]; !internal {
						continue
					}
					if _, seen := visited[call.Target]; seen {
						continue
					}
					visited[call.Target] = struct{}{}
					chain = append(chain, call.Target)
					walk(call.Target, depth+1)
				}
			}
			walk(handler, 0)
		}
		if len(chain) == 0 && handler == event.Handler {
			unresolved++
		}
		scope := "module"
		if event.Scope != nil && *event.Scope != "" {
			scope = *event.Scope
		}
		grouped[scope] = append(grouped[scope], UseCase{Name: useCaseName(event), Actor: event.Actor, Component: event.Component, Event: event.Event, Handler: handler, CallChain: chain})
	}
	scopes := make([]string, 0, len(grouped))
	for scope := range grouped {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	result := UseCaseDiagramBundle{Statistics: map[string]int{"diagram_count": len(scopes), "use_case_count": len(events), "unresolved_handler_count": unresolved}}
	for _, scope := range scopes {
		result.Diagrams = append(result.Diagrams, UseCaseDiagram{Name: safeUseCaseName(scope), Scope: scope, UseCases: grouped[scope]})
	}
	return result, nil
}

func useCaseName(event commonir.InputEvent) string {
	if event.Label != nil && strings.TrimSpace(*event.Label) != "" {
		return strings.TrimSpace(*event.Label)
	}
	name := event.Handler
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	for _, prefix := range []string{"on_", "handle_", "handle", "do_"} {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	for _, suffix := range []string{"_clicked", "_click", "_pressed", "_press", "_triggered", "_changed"} {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
			break
		}
	}
	words := []string{}
	for _, word := range strings.Split(strings.Trim(name, "_"), "_") {
		if word != "" {
			words = append(words, word)
		}
	}
	if len(words) == 0 {
		return "User action"
	}
	value := strings.Join(words, " ")
	return strings.ToUpper(value[:1]) + value[1:]
}

func safeUseCaseName(value string) string {
	name := strings.Trim(safeUseCaseChars.ReplaceAllString(value, "_"), "_.-")
	if name == "" {
		return "use_cases"
	}
	return name
}
