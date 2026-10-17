package renderers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

func RenderStateDiagramMermaid(diagram generators.StateDiagram) string {
	lines := []string{"stateDiagram-v2", fmt.Sprintf("    %%%% %s.%s : %s", diagram.Owner, diagram.StateVariable, diagram.StateType)}
	if diagram.InitialState != nil && *diagram.InitialState != "" {
		lines = append(lines, "    [*] --> "+*diagram.InitialState)
	}
	hasAny := false
	for _, transition := range diagram.Transitions {
		if transition.Source == "*" {
			hasAny = true
			break
		}
	}
	if hasAny {
		lines = append(lines, `    state "Any current state" as __any`)
	}
	for _, transition := range diagram.Transitions {
		source := transition.Source
		if source == "*" {
			source = "__any"
		}
		line := fmt.Sprintf("    %s --> %s", source, transition.Target)
		pieces := []string{}
		if transition.Event != nil && *transition.Event != "" {
			pieces = append(pieces, *transition.Event)
		}
		if transition.Condition != nil && *transition.Condition != "" {
			pieces = append(pieces, "["+*transition.Condition+"]")
		}
		if len(pieces) > 0 {
			line += ": " + strings.Join(pieces, " ")
		}
		lines = append(lines, line)
	}
	for _, terminal := range diagram.TerminalStates {
		lines = append(lines, "    "+terminal+" --> [*]")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderTimingChartMermaid(chart generators.TimingChart) string {
	targetSet := map[string]struct{}{}
	for _, event := range chart.Events {
		if event.Target != nil && *event.Target != "" {
			targetSet[*event.Target] = struct{}{}
		}
	}
	targets := make([]string, 0, len(targetSet))
	for target := range targetSet {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	participants := map[string]string{chart.Owner: "p0"}
	lines := []string{"sequenceDiagram", fmt.Sprintf("    participant p0 as \"%s\"", escapeText(chart.Owner))}
	for i, target := range targets {
		participants[target] = fmt.Sprintf("p%d", i+1)
		lines = append(lines, fmt.Sprintf("    participant %s as \"%s\"", participants[target], escapeText(target)))
	}
	if chart.IsAsync {
		lines = append(lines, "    Note over p0: async flow")
	}
	for _, event := range chart.Events {
		target := ""
		if event.Target != nil {
			target = *event.Target
		}
		targetID := participants[target]
		detail := event.Kind
		if event.Target != nil && *event.Target != "" {
			detail = *event.Target
		}
		if event.Detail != nil && *event.Detail != "" {
			detail = *event.Detail
		}
		detail = escapeText(detail)
		switch event.Kind {
		case "call":
			if targetID != "" {
				lines = append(lines, fmt.Sprintf("    p0->>%s: %d. call %s", targetID, event.Order, detail))
			} else {
				lines = append(lines, fmt.Sprintf("    Note over p0: %d. call %s", event.Order, detail))
			}
		case "await":
			if targetID != "" {
				lines = append(lines, fmt.Sprintf("    p0->>%s: %d. await %s", targetID, event.Order, detail), fmt.Sprintf("    %s-->>p0: resume", targetID))
			} else {
				lines = append(lines, fmt.Sprintf("    Note over p0: %d. await %s", event.Order, detail))
			}
		case "parallel_start":
			if targetID != "" {
				lines = append(lines, fmt.Sprintf("    p0-)%s: %d. parallel start %s", targetID, event.Order, detail))
			} else {
				lines = append(lines, fmt.Sprintf("    Note over p0: %d. parallel_start %s", event.Order, detail))
			}
		case "parallel_join":
			lines = append(lines, fmt.Sprintf("    Note over p0: %d. join %s", event.Order, detail))
		case "wait":
			lines = append(lines, fmt.Sprintf("    Note over p0: %d. wait %s", event.Order, detail))
		case "sync_wait":
			lines = append(lines, fmt.Sprintf("    Note over p0: %d. sync wait %s", event.Order, detail))
		case "sync":
			lines = append(lines, fmt.Sprintf("    Note over p0: %d. sync %s", event.Order, detail))
		case "timer", "callback":
			if targetID != "" {
				lines = append(lines, fmt.Sprintf("    p0-)%s: %d. %s %s", targetID, event.Order, event.Kind, detail))
			} else {
				lines = append(lines, fmt.Sprintf("    Note over p0: %d. %s %s", event.Order, event.Kind, detail))
			}
		case "periodic":
			lines = append(lines, fmt.Sprintf("    Note over p0: %d. periodic %s", event.Order, detail))
		default:
			lines = append(lines, fmt.Sprintf("    Note over p0: %d. %s %s", event.Order, event.Kind, detail))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUseCaseDiagramMermaid(diagram generators.UseCaseDiagram) string {
	actorSet := map[string]struct{}{}
	for _, useCase := range diagram.UseCases {
		actorSet[useCase.Actor] = struct{}{}
	}
	actors := make([]string, 0, len(actorSet))
	for actor := range actorSet {
		actors = append(actors, actor)
	}
	sort.Strings(actors)
	ids := map[string]string{}
	lines := []string{"flowchart LR"}
	for i, actor := range actors {
		ids[actor] = fmt.Sprintf("actor%d", i+1)
		lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", ids[actor], escapeText(actor)))
	}
	for i, useCase := range diagram.UseCases {
		ident := fmt.Sprintf("uc%d", i+1)
		lines = append(lines, fmt.Sprintf("    %s([\"%s\"])", ident, escapeText(useCase.Name)))
		relation := escapeText(useCase.Component + " / " + useCase.Event)
		lines = append(lines, fmt.Sprintf("    %s -->|\"%s\"| %s", ids[useCase.Actor], relation, ident))
		chain := append([]string{useCase.Handler}, useCase.CallChain...)
		lines = append(lines, "    %% handler-chain: "+escapeText(strings.Join(chain, " -> ")))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderDeploymentDiagramMermaid(diagram generators.DeploymentDiagram) string {
	ordered := append([]generators.DeploymentNode(nil), diagram.Nodes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	ids := map[string]string{}
	lines := []string{"flowchart LR"}
	for i, node := range ordered {
		ids[node.ID] = fmt.Sprintf("n%d", i)
		lines = append(lines, deploymentNodeLine(ids[node.ID], deploymentLabel(node), node.Kind))
	}
	for _, connection := range diagram.Connections {
		if ids[connection.Source] == "" || ids[connection.Target] == "" {
			continue
		}
		arrow := "-->"
		if connection.Confidence == "inferred" {
			arrow = "-.->"
		}
		relation := escapeText(fmt.Sprintf("%s [%s]", connection.Relation, connection.Confidence))
		lines = append(lines, fmt.Sprintf("    %s %s|\"%s\"| %s", ids[connection.Source], arrow, relation, ids[connection.Target]))
	}
	lines = append(lines, "    classDef confirmed stroke-width:2px;", "    classDef inferred stroke-dasharray:5 5;", "    classDef unknown stroke-dasharray:2 4;")
	for _, node := range ordered {
		confidence := string(node.Confidence)
		if confidence != "confirmed" && confidence != "inferred" && confidence != "unknown" {
			confidence = "unknown"
		}
		lines = append(lines, fmt.Sprintf("    class %s %s;", ids[node.ID], confidence))
	}
	return strings.Join(lines, "\n") + "\n"
}

func deploymentLabel(node generators.DeploymentNode) string {
	label := node.Label
	detail := node.Metadata["image"]
	if detail == "" {
		detail = node.Metadata["images"]
	}
	if detail != "" {
		label += "\\n" + detail
	}
	if node.Environment != "" {
		label += "\\n[" + node.Environment + "]"
	}
	return label
}

func deploymentNodeLine(id, label, kind string) string {
	text := escapeText(label)
	switch kind {
	case "database":
		return fmt.Sprintf("    %s[(\"%s\")]", id, text)
	case "external-dependency", "external-image":
		return fmt.Sprintf("    %s{{{{\"%s\"}}}}", id, text)
	case "ingress":
		return fmt.Sprintf("    %s([\"%s\"])", id, text)
	case "workload", "container-image":
		return fmt.Sprintf("    %s[[\"%s\"]]", id, text)
	default:
		return fmt.Sprintf("    %s[\"%s\"]", id, text)
	}
}
