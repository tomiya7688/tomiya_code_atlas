package renderers

import (
	"fmt"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

func RenderSequenceDiagramMermaid(diagram generators.SequenceDiagram) string {
	lines := []string{"sequenceDiagram"}
	ids := map[string]string{}
	participants := sequenceRenderingParticipants(diagram)
	for i, participant := range participants {
		ids[participant] = fmt.Sprintf("p%d", i)
		lines = append(lines, fmt.Sprintf("    participant %s as %s", ids[participant], sequenceLabel(participant)))
	}
	for _, message := range diagram.Messages {
		if ids[message.Caller] == "" {
			continue
		}
		if message.Kind == generators.SequenceDepthTruncated {
			lines = append(lines, fmt.Sprintf("    Note over %s: %s", ids[message.Caller], sequenceLabel(message.Label)))
			continue
		}
		if ids[message.Callee] == "" {
			continue
		}
		arrow := "->>"
		label := message.Label
		switch message.Kind {
		case generators.SequenceCallMessage:
			// Synchronous calls use the default arrow.
		case generators.SequenceReturnMessage:
			arrow = "-->>"
		case generators.SequenceAsyncCallMessage:
			arrow = "-)"
		case generators.SequenceRecursiveCall:
			label = "[recursive] " + label
		default:
			continue
		}
		lines = append(lines, fmt.Sprintf("    %s%s%s: %s", ids[message.Caller], arrow, ids[message.Callee], sequenceLabel(label)))
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
	for i, participant := range communicationRenderingParticipants(diagram) {
		ids[participant] = fmt.Sprintf("p%d", i)
		lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", ids[participant], communicationLabel(participant)))
	}
	for _, message := range diagram.Messages {
		if ids[message.Source] == "" || ids[message.Target] == "" {
			continue
		}
		label := message.Label
		switch message.Kind {
		case generators.SequenceRecursiveCall:
			label = "[recursive] " + label
		case generators.SequenceDepthTruncated:
			label = "[depth truncated] " + label
		}
		text := communicationLabel(message.Order + ": " + label)
		lines = append(lines, fmt.Sprintf("    %s -->|\"%s\"| %s", ids[message.Source], text, ids[message.Target]))
	}
	return strings.Join(lines, "\n") + "\n"
}

//	{
//	  責務: [sequenceRenderingParticipants: messageから省略対象を除いたparticipantを順序付きで収集する]
//	}
func sequenceRenderingParticipants(diagram generators.SequenceDiagram) []string {
	participants := append([]string(nil), diagram.Participants...)
	seen := make(map[string]struct{}, len(participants))
	for _, participant := range participants {
		seen[participant] = struct{}{}
	}
	for _, message := range diagram.Messages {
		candidates := []string{message.Caller}
		if message.Kind != generators.SequenceDepthTruncated {
			candidates = append(candidates, message.Callee)
		}
		for _, participant := range candidates {
			appendRendererParticipant(&participants, seen, participant)
		}
	}
	return participants
}

//	{
//	  責務: [communicationRenderingParticipants: messageに現れる全participantを順序付きで収集する]
//	}
func communicationRenderingParticipants(diagram generators.CommunicationDiagram) []string {
	participants := append([]string(nil), diagram.Participants...)
	seen := make(map[string]struct{}, len(participants))
	for _, participant := range participants {
		seen[participant] = struct{}{}
	}
	for _, message := range diagram.Messages {
		for _, participant := range []string{message.Source, message.Target} {
			appendRendererParticipant(&participants, seen, participant)
		}
	}
	return participants
}

//	{
//	  責務: [appendRendererParticipant: 空名と既出名を除外してparticipantを一件追加する]
//	}
func appendRendererParticipant(participants *[]string, seen map[string]struct{}, participant string) {
	if participant == "" {
		return
	}
	if _, exists := seen[participant]; exists {
		return
	}
	seen[participant] = struct{}{}
	*participants = append(*participants, participant)
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
