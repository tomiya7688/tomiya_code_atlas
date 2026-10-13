package renderers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
)

func RenderCallGraphMermaid(graph generators.CallGraph, direction string) string {
	if direction == "" {
		direction = "LR"
	}
	lines := []string{"flowchart " + direction}
	nodes := append([]string(nil), graph.Nodes...)
	sort.Strings(nodes)
	for _, node := range nodes {
		lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", callNodeID(node), node))
	}
	for _, edge := range graph.Edges {
		lines = append(lines, fmt.Sprintf("    %s --> %s", callNodeID(edge.Caller), callNodeID(edge.Callee)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func callNodeID(name string) string { return "n_" + nonNodeID.ReplaceAllString(name, "_") }

var visibilityPrefix = map[string]string{"public": "+", "protected": "#", "internal": "~", "private": "-", "unspecified": "~"}

//	{
//	  責務: [RenderClassDiagramMermaid: class/member/relation logical modelをMermaid classDiagramへ直列化する]
//	}
func RenderClassDiagramMermaid(diagram generators.ClassDiagram) string {
	lines := []string{"classDiagram", "direction LR"}
	ids := map[string]string{}
	for _, node := range diagram.Nodes {
		ids[node.Name] = hashedID("c_", node.Name)
	}
	for _, node := range diagram.Nodes {
		id := ids[node.Name]
		label := escapeQuote(node.Name)
		if len(node.Members) == 0 {
			lines = append(lines, fmt.Sprintf("    class %s[\"%s\"]", id, label))
			continue
		}
		lines = append(lines, fmt.Sprintf("    class %s[\"%s\"] {", id, label))
		for _, member := range node.Members {
			prefix := visibilityPrefix[member.Visibility]
			if prefix == "" {
				prefix = "~"
			}
			lines = append(lines, fmt.Sprintf("        %s%s(%s)", prefix, member.Name, strings.Join(member.Parameters, ", ")))
		}
		lines = append(lines, "    }")
	}
	for _, relation := range diagram.Relations {
		source := ids[relation.Source]
		if source == "" {
			source = hashedID("c_", relation.Source)
		}
		target := ids[relation.Target]
		if target == "" {
			target = hashedID("c_", relation.Target)
		}
		if relation.Kind == "inheritance" {
			lines = append(lines, fmt.Sprintf("    %s <|-- %s", target, source))
		} else {
			lines = append(lines, fmt.Sprintf("    %s --> %s : uses", source, target))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

//	{
//	  責務: [RenderSequenceDiagramMermaid: ordered participant/message modelをMermaid sequenceDiagramへ直列化する]
//	}
