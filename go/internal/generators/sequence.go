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

const (
	SequenceCallMessage      = "call"
	SequenceAsyncCallMessage = "async_call"
	SequenceReturnMessage    = "return"
	SequenceRecursiveCall    = "recursive_call"
	SequenceDepthTruncated   = "depth_truncated"
)

//	{
//	  責務: [SequenceDiagram: ordered message、participants、系列境界参照を保持する]
//	  フィールド: [Name: 図名, Participants: 表示順のparticipant, Messages: ordered call, References: 続きのdiagram参照]
//	}
type SequenceDiagram struct {
	Name         string
	Participants []string
	Messages     []SequenceMessage
	References   []DiagramReference
}

type SequenceDiagramBundle struct {
	Diagrams   []SequenceDiagram
	Statistics map[string]any
	Placements []OutputPlacement
}

//	{
//	  責務: [BuildSequenceDiagramBundle: ordered Common IR callからbounded sequence diagramsを作る]
//	  処理: [1: call sequenceを解決, 2: 内部relationをpartition, 3: rootごとにrecursionとdepth truncationを区別した順序付きmessageを作る]
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
	partition, err := analyzers.PartitionGraph(relation, fanInThreshold, analyzers.DefaultMinChildSeriesSize)
	if err != nil {
		return SequenceDiagramBundle{}, err
	}
	sharedOutputs := map[string]string{}
	for _, shared := range partition.Shared {
		sharedOutputs[shared] = stableOutputName("shared", shared, "sequence")
	}
	seriesOutputs := map[string]string{}
	//	{
	//	  背景: [shared partitionは個別のtraversal pathを持たない]
	//	  処理: [グラフ上のcycleに含まれるedgeをrecursive messageとして識別する]
	//	}
	cycleEdges := cycleEdges(graph.Cycles())
	incoming := map[string]map[string]struct{}{}
	for _, node := range graph.Nodes() {
		incoming[node] = map[string]struct{}{}
	}
	for _, edge := range graph.Edges() {
		incoming[edge.Callee][edge.Caller] = struct{}{}
	}
	result := SequenceDiagramBundle{Statistics: partitionStatistics(partition)}
	rootOutputs := map[string]string{}
	continuationOutputs := map[string]string{}
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
		seriesStart := len(result.Diagrams)
		for _, root := range roots {
			name := stableOutputName(fmt.Sprintf("series_%d", i+1), root, "sequence")
			diagram := buildSequenceDiagram(root, name, sequences, selected, maxDepth, resolvedOptions)
			result.Diagrams = append(result.Diagrams, diagram)
			rootOutputs[root] = name
			result.Placements = append(result.Placements, regularPlacement(name, partition, i))
		}
		// 深さ上限で省いたcallのtargetを新しい図のrootにし、全internal edgeを有限回で展開します。
		queue := sequenceContinuationTargets(result.Diagrams[seriesStart:], sequences)
		continuationNumber := 0
		for queueIndex := 0; queueIndex < len(queue); queueIndex++ {
			root := queue[queueIndex]
			if rootOutputs[root] != "" || len(sequences[root]) == 0 {
				continue
			}
			continuationNumber++
			name := stableOutputName(fmt.Sprintf("series_%d_continuation_%d", i+1, continuationNumber), root, "sequence")
			diagram := buildSequenceDiagram(root, name, sequences, selected, maxDepth, resolvedOptions)
			result.Diagrams = append(result.Diagrams, diagram)
			result.Placements = append(result.Placements, regularPlacement(name, partition, i))
			rootOutputs[root] = name
			continuationOutputs[root] = name
			queue = append(queue, sequenceContinuationTargets([]SequenceDiagram{diagram}, sequences)...)
		}
		for _, node := range series {
			output := rootOutputs[node]
			if output == "" {
				output = continuationOutputs[node]
			}
			if output == "" {
				for _, diagram := range result.Diagrams[seriesStart:] {
					if containsString(diagram.Participants, node) {
						output = diagram.Name
						break
					}
				}
			}
			if output != "" {
				seriesOutputs[node] = output
			}
		}
	}
	for diagramIndex := range result.Diagrams {
		diagram := &result.Diagrams[diagramIndex]
		if diagramIndex >= len(result.Placements) || result.Placements[diagramIndex].Shared {
			continue
		}
		seriesIndex := indexOfSeriesRoot(partition, result.Placements[diagramIndex].SeriesRoot)
		if seriesIndex < 0 {
			continue
		}
		references := partitionReferences(partition, seriesIndex, sharedOutputs, seriesOutputs)
		for _, message := range diagram.Messages {
			if message.Kind != SequenceDepthTruncated {
				continue
			}
			output := continuationOutputs[message.Callee]
			if output == "" {
				output = rootOutputs[message.Callee]
			}
			if output != "" && output != diagram.Name {
				references = append(references, DiagramReference{Node: message.Callee, OutputName: output})
			}
		}
		diagram.References = filterDiagramReferences(references, diagram.Participants)
	}
	for _, shared := range partition.Shared {
		messages := []SequenceMessage{}
		emitted := map[string]struct{}{}
		callers := sortedSet(incoming[shared])
		for _, caller := range callers {
			label := firstCallLabel(caller, shared, sequences)
			kind := SequenceCallMessage
			if cycleEdges[[2]string{caller, shared}] {
				kind = SequenceRecursiveCall
			}
			_, sharedIsInternal := sequences[shared]
			if appendSequenceCall(&messages, emitted, caller, shared, label, 0, kind, resolvedOptions) && resolvedOptions.ShowReturns && sharedIsInternal {
				messages = append(messages, SequenceMessage{Caller: shared, Callee: caller, Label: "return", Depth: 0, Kind: SequenceReturnMessage})
			}
		}
		for _, call := range sequences[shared] {
			kind := SequenceCallMessage
			if cycleEdges[[2]string{shared, call.Target}] {
				kind = SequenceRecursiveCall
			}
			if appendSequenceCall(&messages, emitted, shared, call.Target, call.Raw, 1, kind, resolvedOptions) && resolvedOptions.ShowReturns {
				if _, internal := sequences[call.Target]; internal {
					messages = append(messages, SequenceMessage{Caller: call.Target, Callee: shared, Label: "return", Depth: 1, Kind: SequenceReturnMessage})
				}
			}
		}
		name := stableOutputName("shared", shared, "sequence")
		participants := sequenceParticipants(shared, messages)
		result.Diagrams = append(result.Diagrams, SequenceDiagram{Name: name, Participants: participants, Messages: messages, References: filterDiagramReferences(sharedDiagramReferences(shared, partition, sharedOutputs, seriesOutputs), participants)})
		result.Placements = append(result.Placements, sharedPlacement(name, shared))
	}
	return result, nil
}

//	{
//	  責務: [buildSequenceDiagram: 指定rootから深さ制限付きsequence図を構築する]
//	}
func buildSequenceDiagram(root, name string, sequences map[string][]analyzers.ResolvedCall, selected map[string]struct{}, maxDepth int, options SequenceDiagramOptions) SequenceDiagram {
	messages := expandSequence(root, sequences, selected, maxDepth, options)
	return SequenceDiagram{Name: name, Participants: sequenceParticipants(root, messages), Messages: messages}
}

//	{
//	  責務: [sequenceContinuationTargets: 深さ省略callから追加図のrootを順序付きで集める]
//	}
func sequenceContinuationTargets(diagrams []SequenceDiagram, sequences map[string][]analyzers.ResolvedCall) []string {
	targets := []string{}
	seen := map[string]struct{}{}
	for _, diagram := range diagrams {
		for _, message := range diagram.Messages {
			if message.Kind != SequenceDepthTruncated || len(sequences[message.Callee]) == 0 {
				continue
			}
			if _, exists := seen[message.Callee]; exists {
				continue
			}
			seen[message.Callee] = struct{}{}
			targets = append(targets, message.Callee)
		}
	}
	return targets
}

//	{
//	  責務: [containsString: 図に含まれるparticipantを検索する]
//	}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

//	{
//	  責務: [indexOfSeriesRoot: placementのseries rootからpartition indexを検索する]
//	}
func indexOfSeriesRoot(partition analyzers.GraphPartition, root string) int {
	for index, seriesRoot := range partition.SeriesRoots {
		if seriesRoot == root {
			return index
		}
	}
	return -1
}

//	{
//	  責務: [expandSequence: recursionをcycle-closing messageとして残し、深さ制限で省く展開を明示する]
//	}
func expandSequence(root string, sequences map[string][]analyzers.ResolvedCall, allowed map[string]struct{}, maxDepth int, options SequenceDiagramOptions) []SequenceMessage {
	messages := []SequenceMessage{}
	emitted := map[string]struct{}{}
	var walk func(string, int, map[string]struct{})
	walk = func(caller string, depth int, active map[string]struct{}) {
		if depth >= maxDepth {
			appendDepthTruncations(&messages, caller, depth, sequences[caller], sequences, allowed, active, emitted, options)
			return
		}
		for _, call := range sequences[caller] {
			kind := SequenceCallMessage
			_, cycleClosing := active[call.Target]
			if cycleClosing {
				kind = SequenceRecursiveCall
			}
			if !appendSequenceCall(&messages, emitted, caller, call.Target, call.Raw, depth, kind, options) {
				continue
			}
			_, internal := sequences[call.Target]
			if cycleClosing {
				if options.ShowReturns && internal {
					messages = append(messages, SequenceMessage{Caller: call.Target, Callee: caller, Label: "return", Depth: depth, Kind: SequenceReturnMessage})
				}
				continue
			}
			if _, permitted := allowed[call.Target]; permitted && internal {
				next := make(map[string]struct{}, len(active)+1)
				for name := range active {
					next[name] = struct{}{}
				}
				next[call.Target] = struct{}{}
				walk(call.Target, depth+1, next)
			}
			if options.ShowReturns && internal {
				messages = append(messages, SequenceMessage{Caller: call.Target, Callee: caller, Label: "return", Depth: depth, Kind: SequenceReturnMessage})
			}
		}
	}
	walk(root, 0, map[string]struct{}{root: {}})
	return messages
}

//	{
//	  背景: [depth上限を越えるtargetの処理は展開していない]
//	  責務: [appendDepthTruncations: 未展開callを注記し、実行されていないcall/returnを描かない]
//	}
func appendDepthTruncations(messages *[]SequenceMessage, caller string, depth int, calls []analyzers.ResolvedCall, sequences map[string][]analyzers.ResolvedCall, allowed, active map[string]struct{}, emitted map[string]struct{}, options SequenceDiagramOptions) {
	for _, call := range calls {
		if _, activeTarget := active[call.Target]; activeTarget {
			if !appendSequenceCall(messages, emitted, caller, call.Target, call.Raw, depth, SequenceRecursiveCall, options) {
				continue
			}
			_, internal := sequences[call.Target]
			if options.ShowReturns && internal {
				*messages = append(*messages, SequenceMessage{Caller: call.Target, Callee: caller, Label: "return", Depth: depth, Kind: SequenceReturnMessage})
			}
			continue
		}
		if _, inSeries := allowed[call.Target]; !inSeries {
			if appendSequenceCall(messages, emitted, caller, call.Target, call.Raw, depth, SequenceCallMessage, options) {
				_, internal := sequences[call.Target]
				if options.ShowReturns && internal {
					*messages = append(*messages, SequenceMessage{Caller: call.Target, Callee: caller, Label: "return", Depth: depth, Kind: SequenceReturnMessage})
				}
			}
			continue
		}
		appendSequenceCall(messages, emitted, caller, call.Target, "Further calls omitted: "+call.Raw, depth, SequenceDepthTruncated, options)
	}
}

//	{
//	  責務: [appendSequenceCall: logical messageを重複規則に従って順序付きsequenceへ追加する]
//	}
func appendSequenceCall(messages *[]SequenceMessage, emitted map[string]struct{}, caller, callee, label string, depth int, kind string, options SequenceDiagramOptions) bool {
	key := caller + "\x00" + callee + "\x00" + label + "\x00" + kind
	if !options.ShowDuplicateCalls {
		if _, exists := emitted[key]; exists {
			return false
		}
	}
	emitted[key] = struct{}{}
	*messages = append(*messages, SequenceMessage{Caller: caller, Callee: callee, Label: label, Depth: depth, Kind: kind})
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
		participants := []string{message.Caller}
		participants = append(participants, message.Callee)
		for _, name := range participants {
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

//	{
//	  責務: [CommunicationMessage: 呼び出し元・呼び出し先・順番とcallの意味を保持する]
//	}
type CommunicationMessage struct {
	Source, Target, Order, Label string
	Kind                         string
}

//	{
//	  責務: [CommunicationDiagram: 順番付きcallと系列境界参照を保持する]
//	  フィールド: [Name: 図名, Participants: 表示participant, Messages: call順序と関係, References: 続きのdiagram参照]
//	}
type CommunicationDiagram struct {
	Name         string
	Participants []string
	Messages     []CommunicationMessage
	References   []DiagramReference
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
			if isInteractionDiagramMessage(message.Kind) {
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
			messages = append(messages, CommunicationMessage{Source: communicationParticipant(message.Caller), Target: communicationParticipant(message.Callee), Order: orders[i], Label: message.Label, Kind: message.Kind})
		}
		result.Diagrams = append(result.Diagrams, CommunicationDiagram{Name: sequence.Name, Participants: communicationParticipants(sequence.Participants, messages), Messages: messages, References: append([]DiagramReference(nil), sequence.References...)})
	}
	return result, nil
}

//	{
//	  責務: [isInteractionDiagramMessage: interaction diagramに表示するcallと省略注記を判定する]
//	}
func isInteractionDiagramMessage(kind string) bool {
	return kind == SequenceCallMessage || kind == SequenceAsyncCallMessage || kind == SequenceRecursiveCall || kind == SequenceDepthTruncated
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
