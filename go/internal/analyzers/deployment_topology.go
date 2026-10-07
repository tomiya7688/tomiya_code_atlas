package analyzers

import "sort"

// Confidence records how directly a deployment fact was observed.
type Confidence string

const (
	ConfidenceUnknown   Confidence = "unknown"
	ConfidenceInferred  Confidence = "inferred"
	ConfidenceConfirmed Confidence = "confirmed"
)

// DeploymentNode is a renderer-neutral logical deployment entity.
type DeploymentNode struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Kind        string            `json:"kind"`
	Confidence  Confidence        `json:"confidence"`
	Environment string            `json:"environment,omitempty"`
	Source      string            `json:"source,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DeploymentConnection is a logical relation between deployment nodes.
type DeploymentConnection struct {
	Source     string     `json:"source"`
	Target     string     `json:"target"`
	Relation   string     `json:"relation"`
	Confidence Confidence `json:"confidence"`
}

// DeploymentTopology contains normalized deployment facts only.
type DeploymentTopology struct {
	Nodes       []DeploymentNode       `json:"nodes"`
	Connections []DeploymentConnection `json:"connections"`
}

// MergeDeploymentTopologies deduplicates facts and keeps the most certain one.
func MergeDeploymentTopologies(topologies ...DeploymentTopology) DeploymentTopology {
	nodes := make(map[string]DeploymentNode)
	connections := make(map[deploymentConnectionKey]DeploymentConnection)
	for _, topology := range topologies {
		for _, node := range topology.Nodes {
			current, exists := nodes[node.ID]
			if !exists || confidenceRank(node.Confidence) > confidenceRank(current.Confidence) {
				nodes[node.ID] = cloneDeploymentNode(node)
			}
		}
		for _, connection := range topology.Connections {
			key := deploymentConnectionKey{connection.Source, connection.Target, connection.Relation}
			current, exists := connections[key]
			if !exists || confidenceRank(connection.Confidence) > confidenceRank(current.Confidence) {
				connections[key] = connection
			}
		}
	}
	result := emptyDeploymentTopology()
	for _, node := range nodes {
		result.Nodes = append(result.Nodes, node)
	}
	for _, connection := range connections {
		result.Connections = append(result.Connections, connection)
	}
	sort.Slice(result.Nodes, func(i, j int) bool { return result.Nodes[i].ID < result.Nodes[j].ID })
	sort.Slice(result.Connections, func(i, j int) bool {
		left, right := result.Connections[i], result.Connections[j]
		if left.Source != right.Source {
			return left.Source < right.Source
		}
		if left.Target != right.Target {
			return left.Target < right.Target
		}
		return left.Relation < right.Relation
	})
	return result
}

// DeploymentTopologyFromComponentGraph converts source dependencies to inferred placement facts.
func DeploymentTopologyFromComponentGraph(graph ComponentDependencyGraph) DeploymentTopology {
	result := emptyDeploymentTopology()
	external := make(map[string]struct{})
	for _, name := range graph.ExternalNodes() {
		external[name] = struct{}{}
	}
	for _, name := range graph.Nodes() {
		if _, isExternal := external[name]; isExternal {
			label := name
			if len(name) >= len("external:") && name[:len("external:")] == "external:" {
				label = name[len("external:"):]
			}
			result.Nodes = append(result.Nodes, DeploymentNode{ID: name, Label: label, Kind: "external-dependency", Confidence: ConfidenceInferred, Source: "source imports"})
		} else {
			result.Nodes = append(result.Nodes, DeploymentNode{ID: "component:" + name, Label: name, Kind: "application-component", Confidence: ConfidenceInferred, Source: "source imports"})
		}
	}
	for _, edge := range graph.Edges() {
		source, target := "component:"+edge.Caller, "component:"+edge.Callee
		if _, isExternal := external[edge.Caller]; isExternal {
			source = edge.Caller
		}
		if _, isExternal := external[edge.Callee]; isExternal {
			target = edge.Callee
		}
		result.Connections = append(result.Connections, DeploymentConnection{Source: source, Target: target, Relation: "imports", Confidence: ConfidenceInferred})
	}
	return result
}

type deploymentConnectionKey struct{ source, target, relation string }

func confidenceRank(confidence Confidence) int {
	switch confidence {
	case ConfidenceConfirmed:
		return 2
	case ConfidenceInferred:
		return 1
	default:
		return 0
	}
}

func cloneDeploymentNode(node DeploymentNode) DeploymentNode {
	if node.Metadata != nil {
		metadata := make(map[string]string, len(node.Metadata))
		for key, value := range node.Metadata {
			metadata[key] = value
		}
		node.Metadata = metadata
	}
	return node
}
