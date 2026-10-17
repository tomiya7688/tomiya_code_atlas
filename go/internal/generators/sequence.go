package generators

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [SequenceDiagramOptions: sequence logical model生成時の重複call・return表示規則を保持する]
//	}
type SequenceDiagramOptions struct {
	ShowDuplicateCalls bool
	ShowReturns        bool
}

func DefaultSequenceDiagramOptions() SequenceDiagramOptions {
	return SequenceDiagramOptions{ShowDuplicateCalls: true, ShowReturns: false}
}

type SequenceMessage struct {
	Caller, Callee, Label string
	Depth                 int
	Kind                  string
}

type SequenceDiagram struct {
	Name         string
	Participants []string
	Messages     []SequenceMessage
}

type SequenceDiagramBundle struct {
	Diagrams   []SequenceDiagram
	Statistics map[string]any
	Placements []OutputPlacement
}

//	{
//	  責務: [BuildSequenceDiagramBundle: ordered Common IR callからbounded sequence diagramsを作る]
//	  処理: [1: call sequenceを解決, 2: 内部relationをpartition, 3: cycle edgeを展開対象外にする, 4: rootごとに順序付きmessageを作る]
//	}
func BuildSequenceDiagramBundle(module commonir.Module, fanInThreshold, maxDepth int, options *SequenceDiagramOptions) (SequenceDiagramBundle, error) {
	if maxDepth < 0 {
		return SequenceDiagramBundle{}, fmt.Errorf("max depth must be >= 0")
	}
	resolvedOptions := DefaultSequenceDiagramOptions()
	if options != nil {
		resolvedOptions = *options
	}
	sequences := analyzers.ResolveCallSequences(module)
	graph := analyzers.BuildSequenceRelationGraph(module, sequences)
	relation, err := graph.RelationGraph()
	if err != nil {
		return SequenceDiagramBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, 3)
	if err != nil {
		return SequenceDiagramBundle{}, err
	}
	blocked := cycleEdges(graph.Cycles())
	incoming := map[string]map[string]struct{}{}
	for _, node := range graph.Nodes() {
		incoming[node] = map[string]struct{}{}
	}
	for _, edge := range graph.Edges() {
		incoming[edge.Callee][edge.Caller] = struct{}{}
	}
	result := SequenceDiagramBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		selected := stringSet(series)
		roots := []string{}
		for _, node := range series {
			hasIncoming := false
			for caller := range incoming[node] {
				if _, ok := selected[caller]; ok {
					hasIncoming = true
					break
				}
			}
			if !hasIncoming {
				roots = append(roots, node)
			}
		}
		if len(roots) == 0 && len(series) > 0 {
			roots = []string{series[0]}
		}
		for _, root := range roots {
			messages := expandSequence(root, sequences, selected, maxDepth, resolvedOptions, blocked)
			name := stableOutputName(fmt.Sprintf("series_%d", i+1), root, "sequence")
			result.Diagrams = append(result.Diagrams, SequenceDiagram{Name: name, Participants: sequenceParticipants(root, messages), Messages: messages})
			result.Placements = append(result.Placements, regularPlacement(name, partition, i))
		}
	}
	for _, shared := range partition.Shared {
		messages := []SequenceMessage{}
		emitted := map[string]struct{}{}
		callers := sortedSet(incoming[shared])
		for _, caller := range callers {
			if blocked[[2]string{caller, shared}] {
				continue
			}
			label := firstCallLabel(caller, shared, sequences)
			if appendSequenceCall(&messages, emitted, caller, shared, label, 0, resolvedOptions) && resolvedOptions.ShowReturns {
				messages = append(messages, SequenceMessage{Caller: shared, Callee: caller, Label: "return", Depth: 0, Kind: "return"})
			}
		}
		for _, call := range sequences[shared] {
			if blocked[[2]string{shared, call.Target}] {
				continue
			}
			if appendSequenceCall(&messages, emitted, shared, call.Target, call.Raw, 1, resolvedOptions) && resolvedOptions.ShowReturns {
				if _, internal := sequences[call.Target]; internal {
					messages = append(messages, SequenceMessage{Caller: call.Target, Callee: shared, Label: "return", Depth: 1, Kind: "return"})
				}
			}
		}
		name := stableOutputName("shared", shared, "sequence")
		result.Diagrams = append(result.Diagrams, SequenceDiagram{Name: name, Participants: sequenceParticipants(shared, messages), Messages: messages})
		result.Placements = append(result.Placements, sharedPlacement(name, shared))
	}
	return result, nil
}

func expandSequence(root string, sequences map[string][]analyzers.ResolvedCall, allowed map[string]struct{}, maxDepth int, options SequenceDiagramOptions, blocked map[[2]string]bool) []SequenceMessage {
	messages := []SequenceMessage{}
	emitted := map[string]struct{}{}
	var walk func(string, int, map[string]struct{})
	walk = func(caller string, depth int, active map[string]struct{}) {
		if depth >= maxDepth {
			return
		}
		for _, call := range sequences[caller] {
			if blocked[[2]string{caller, call.Target}] {
				continue
			}
			if _, seen := active[call.Target]; seen {
				continue
			}
			if !appendSequenceCall(&messages, emitted, caller, call.Target, call.Raw, depth, options) {
				continue
			}
			_, internal := sequences[call.Target]
			if _, permitted := allowed[call.Target]; permitted && internal {
				next := make(map[string]struct{}, len(active)+1)
				for name := range active {
					next[name] = struct{}{}
				}
				next[call.Target] = struct{}{}
				walk(call.Target, depth+1, next)
			}
			if options.ShowReturns && internal {
				messages = append(messages, SequenceMessage{Caller: call.Target, Callee: caller, Label: "return", Depth: depth, Kind: "return"})
			}
		}
	}
	walk(root, 0, map[string]struct{}{root: {}})
	return messages
}

func appendSequenceCall(messages *[]SequenceMessage, emitted map[string]struct{}, caller, callee, label string, depth int, options SequenceDiagramOptions) bool {
	key := caller + "\x00" + callee + "\x00" + label
	if !options.ShowDuplicateCalls {
		if _, exists := emitted[key]; exists {
			return false
		}
	}
	emitted[key] = struct{}{}
	*messages = append(*messages, SequenceMessage{Caller: caller, Callee: callee, Label: label, Depth: depth, Kind: "call"})
	return true
}

func cycleEdges(cycles [][]string) map[[2]string]bool {
	result := map[[2]string]bool{}
	for _, cycle := range cycles {
		for i := 0; i+1 < len(cycle); i++ {
			result[[2]string{cycle[i], cycle[i+1]}] = true
		}
	}
	return result
}

func sequenceParticipants(root string, messages []SequenceMessage) []string {
	ordered := []string{root}
	seen := map[string]struct{}{root: {}}
	for _, message := range messages {
		for _, name := range []string{message.Caller, message.Callee} {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			ordered = append(ordered, name)
		}
	}
	return ordered
}

func firstCallLabel(caller, callee string, sequences map[string][]analyzers.ResolvedCall) string {
	for _, call := range sequences[caller] {
		if call.Target == callee {
			return call.Raw
		}
	}
	parts := strings.Split(callee, ".")
	return parts[len(parts)-1]
}

type CommunicationMessage struct{ Source, Target, Order, Label string }
type CommunicationDiagram struct {
	Name         string
	Participants []string
	Messages     []CommunicationMessage
}
type CommunicationDiagramBundle struct {
	Diagrams   []CommunicationDiagram
	Statistics map[string]any
	Placements []OutputPlacement
}

//	{
//	  責務: [BuildCommunicationDiagramBundle: sequence logical modelを番号付きcommunication diagramへ変換する]
//	}
func BuildCommunicationDiagramBundle(module commonir.Module, fanInThreshold, maxDepth int, showDuplicateCalls bool) (CommunicationDiagramBundle, error) {
	sequenceBundle, err := BuildSequenceDiagramBundle(module, fanInThreshold, maxDepth, &SequenceDiagramOptions{ShowDuplicateCalls: showDuplicateCalls, ShowReturns: false})
	if err != nil {
		return CommunicationDiagramBundle{}, err
	}
	result := CommunicationDiagramBundle{Statistics: sequenceBundle.Statistics, Placements: append([]OutputPlacement(nil), sequenceBundle.Placements...)}
	for _, sequence := range sequenceBundle.Diagrams {
		calls := []SequenceMessage{}
		for _, message := range sequence.Messages {
			if message.Kind == "call" {
				calls = append(calls, message)
			}
		}
		depths := make([]int, len(calls))
		for i, message := range calls {
			depths[i] = message.Depth
		}
		orders := messageOrders(depths)
		messages := make([]CommunicationMessage, 0, len(calls))
		for i, message := range calls {
			messages = append(messages, CommunicationMessage{Source: communicationParticipant(message.Caller), Target: communicationParticipant(message.Callee), Order: orders[i], Label: message.Label})
		}
		result.Diagrams = append(result.Diagrams, CommunicationDiagram{Name: sequence.Name, Participants: communicationParticipants(sequence.Participants, messages), Messages: messages})
	}
	return result, nil
}

func messageOrders(depths []int) []string {
	counters := []int{}
	result := make([]string, 0, len(depths))
	for _, depth := range depths {
		if depth < 0 {
			continue
		}
		for len(counters) <= depth {
			counters = append(counters, 0)
		}
		counters = counters[:depth+1]
		counters[depth]++
		parts := make([]string, len(counters))
		for i, value := range counters {
			parts[i] = fmt.Sprint(value)
		}
		result = append(result, strings.Join(parts, "."))
	}
	return result
}

func communicationParticipant(callable string) string {
	if index := strings.LastIndex(callable, "."); index >= 0 {
		return callable[:index]
	}
	return callable
}

func communicationParticipants(candidates []string, messages []CommunicationMessage) []string {
	ordered := []string{}
	seen := map[string]struct{}{}
	appendName := func(name string) {
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		ordered = append(ordered, name)
	}
	for _, name := range candidates {
		appendName(communicationParticipant(name))
	}
	for _, message := range messages {
		appendName(message.Source)
		appendName(message.Target)
	}
	return ordered
}

func sortedMapKeys(values map[string][]analyzers.ResolvedCall) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
