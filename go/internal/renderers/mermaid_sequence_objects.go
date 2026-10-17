package renderers

import (
	"fmt"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

func RenderSequenceDiagramMermaid(diagram generators.SequenceDiagram) string {
	lines := []string{"sequenceDiagram"}
	ids := map[string]string{}
	for i, participant := range diagram.Participants {
		ids[participant] = fmt.Sprintf("p%d", i)
		lines = append(lines, fmt.Sprintf("    participant %s as %s", ids[participant], sequenceLabel(participant)))
	}
	for _, message := range diagram.Messages {
		arrow := "->>"
		if message.Kind == "return" {
			arrow = "-->>"
		}
		lines = append(lines, fmt.Sprintf("    %s%s%s: %s", ids[message.Caller], arrow, ids[message.Callee], sequenceLabel(message.Label)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func sequenceLabel(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, ":", "-")
}

func RenderCommunicationDiagramMermaid(diagram generators.CommunicationDiagram) string {
	lines := []string{"flowchart LR"}
	ids := map[string]string{}
	for i, participant := range diagram.Participants {
		ids[participant] = fmt.Sprintf("p%d", i)
		lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", ids[participant], communicationLabel(participant)))
	}
	for _, message := range diagram.Messages {
		text := communicationLabel(message.Order + ": " + message.Label)
		lines = append(lines, fmt.Sprintf("    %s -->|\"%s\"| %s", ids[message.Source], text, ids[message.Target]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func communicationLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, `"`, `'`)
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.ReplaceAll(value, "\r", " ")
}

func RenderPackageDiagramMermaid(diagram generators.PackageDiagram) string {
	lines := []string{"flowchart LR"}
	for _, node := range diagram.Nodes {
		lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", packageNodeID(node), node))
	}
	for _, edge := range diagram.Edges {
		lines = append(lines, fmt.Sprintf("    %s --> %s", packageNodeID(edge.Source), packageNodeID(edge.Target)))
	}
	if len(diagram.CycleNodes) > 0 {
		lines = append(lines, "    classDef cycle stroke-width:3px")
		ids := make([]string, 0, len(diagram.CycleNodes))
		for _, node := range diagram.CycleNodes {
			ids = append(ids, packageNodeID(node))
		}
		lines = append(lines, "    class "+strings.Join(ids, ",")+" cycle")
	}
	if len(diagram.IsolatedNodes) > 0 {
		lines = append(lines, "    classDef isolated stroke-dasharray: 5 5")
		ids := make([]string, 0, len(diagram.IsolatedNodes))
		for _, node := range diagram.IsolatedNodes {
			ids = append(ids, packageNodeID(node))
		}
		lines = append(lines, "    class "+strings.Join(ids, ",")+" isolated")
	}
	return strings.Join(lines, "\n") + "\n"
}

func packageNodeID(name string) string { return "pkg_" + nonNodeID.ReplaceAllString(name, "_") }

func RenderComponentDiagramMermaid(diagram generators.ComponentDiagram) string {
	lines := []string{"flowchart LR"}
	for _, node := range diagram.Nodes {
		id := "component_" + nonNodeID.ReplaceAllString(node.Name, "_")
		label := strings.TrimPrefix(node.Name, "external:")
		if node.External {
			lines = append(lines, fmt.Sprintf("    %s([\"%s\"])", id, label))
		} else {
			lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", id, label))
		}
	}
	for _, edge := range diagram.Edges {
		lines = append(lines, fmt.Sprintf("    component_%s --> component_%s", nonNodeID.ReplaceAllString(edge.Source, "_"), nonNodeID.ReplaceAllString(edge.Target, "_")))
	}
	if len(diagram.CycleNodes) > 0 {
		lines = append(lines, "    classDef cycle stroke-width:3px")
		ids := make([]string, 0, len(diagram.CycleNodes))
		for _, node := range diagram.CycleNodes {
			ids = append(ids, "component_"+nonNodeID.ReplaceAllString(node, "_"))
		}
		lines = append(lines, "    class "+strings.Join(ids, ",")+" cycle")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderObjectDiagramMermaid(diagram generators.ObjectDiagram) string {
	lines := []string{"flowchart LR"}
	ids := map[string]string{}
	for i, node := range diagram.Nodes {
		ids[node.Key] = fmt.Sprintf("o%d", i)
		parts := []string{node.Name + " : " + node.TypeName}
		for _, field := range node.Fields {
			parts = append(parts, field.Name+" = "+field.Value)
		}
		for j := range parts {
			parts[j] = objectLabel(parts[j])
		}
		lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", ids[node.Key], strings.Join(parts, "<br/>")))
	}
	for _, reference := range diagram.References {
		if ids[reference.Source] == "" || ids[reference.Target] == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("    %s -->|%s| %s", ids[reference.Source], strings.ReplaceAll(objectLabel(reference.Label), "|", "/"), ids[reference.Target]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func objectLabel(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, `"`, `'`)
}
