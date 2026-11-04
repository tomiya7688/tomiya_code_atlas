package renderers

import (
	"fmt"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

//	{
//	  責務: [RenderClassDiagramPlantUML: renderer-neutral class diagramをPlantUMLへ直列化する]
//	}
func RenderClassDiagramPlantUML(diagram generators.ClassDiagram) string {
	aliases := map[string]string{}
	for _, node := range diagram.Nodes {
		aliases[node.Name] = hashedID("c_", node.Name)
	}
	lines := []string{"@startuml", "left to right direction"}
	for _, node := range diagram.Nodes {
		alias := aliases[node.Name]
		label := escapeQuote(node.Name)
		if node.External {
			lines = append(lines, fmt.Sprintf("class \"%s\" as %s <<external>>", label, alias))
			continue
		}
		lines = append(lines, fmt.Sprintf("class \"%s\" as %s {", label, alias))
		for _, member := range node.Members {
			prefix := visibilityPrefix[member.Visibility]
			if prefix == "" {
				prefix = "~"
			}
			lines = append(lines, fmt.Sprintf("  %s%s(%s)", prefix, member.Name, strings.Join(member.Parameters, ", ")))
		}
		lines = append(lines, "}")
	}
	for _, relation := range diagram.Relations {
		source := aliases[relation.Source]
		if source == "" {
			source = hashedID("c_", relation.Source)
		}
		target := aliases[relation.Target]
		if target == "" {
			target = hashedID("c_", relation.Target)
		}
		if relation.Kind == "inheritance" {
			lines = append(lines, fmt.Sprintf("%s <|-- %s", target, source))
		} else {
			lines = append(lines, fmt.Sprintf("%s --> %s : uses", source, target))
		}
	}
	lines = append(lines, "@enduml")
	return strings.Join(lines, "\n") + "\n"
}

//	{
//	  責務: [RenderSequenceDiagramPlantUML: renderer-neutral sequence diagramをPlantUMLへ直列化する]
//	}
func RenderSequenceDiagramPlantUML(diagram generators.SequenceDiagram) string {
	aliases := map[string]string{}
	lines := []string{"@startuml"}
	for _, participant := range sequenceRenderingParticipants(diagram) {
		aliases[participant] = hashedID("p_", participant)
		lines = append(lines, fmt.Sprintf("participant \"%s\" as %s", escapePlantUMLText(participant), aliases[participant]))
	}
	for _, message := range diagram.Messages {
		caller := aliases[message.Caller]
		callee := aliases[message.Callee]
		if caller == "" {
			continue
		}
		if message.Kind == generators.SequenceDepthTruncated {
			lines = append(lines, fmt.Sprintf("note over %s : %s", caller, escapePlantUMLText(message.Label)))
			continue
		}
		if callee == "" {
			continue
		}
		arrow := "->"
		label := message.Label
		switch message.Kind {
		case generators.SequenceCallMessage:
			// Synchronous calls use the default arrow.
		case generators.SequenceReturnMessage:
			arrow = "-->"
		case generators.SequenceAsyncCallMessage:
			arrow = "->>"
		case generators.SequenceRecursiveCall:
			label = "[recursive] " + label
		default:
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s %s : %s", caller, arrow, callee, escapePlantUMLText(label)))
	}
	lines = append(lines, "@enduml")
	return strings.Join(lines, "\n") + "\n"
}

func escapePlantUMLText(value string) string {
	value = escapeQuote(value)
	return strings.ReplaceAll(value, "\n", " ")
}
