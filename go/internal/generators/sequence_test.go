package generators

import (
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [TestSequenceBundleKeepsRecursiveCalls: self recursionとmutual recursionのcycle-closing callを保持する]
//	}
func TestSequenceBundleKeepsRecursiveCalls(t *testing.T) {
	type expectedMessage struct {
		caller string
		callee string
		kind   string
	}
	tests := []struct {
		name   string
		module commonir.Module
		want   map[expectedMessage]int
	}{
		{
			name: "self recursion",
			module: sequenceTestModule(
				commonir.Entity{Kind: "function", Name: "recurse", CallSequence: []string{"recurse"}},
			),
			want: map[expectedMessage]int{{caller: "recurse", callee: "recurse", kind: SequenceRecursiveCall}: 1},
		},
		{
			name: "mutual recursion",
			module: sequenceTestModule(
				commonir.Entity{Kind: "function", Name: "alpha", CallSequence: []string{"beta"}},
				commonir.Entity{Kind: "function", Name: "beta", CallSequence: []string{"alpha"}},
			),
			want: map[expectedMessage]int{
				{caller: "alpha", callee: "beta", kind: SequenceCallMessage}:   1,
				{caller: "beta", callee: "alpha", kind: SequenceRecursiveCall}: 1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bundle, err := BuildSequenceDiagramBundle(test.module, 3, 8, nil)
			if err != nil {
				t.Fatalf("BuildSequenceDiagramBundle() error = %v", err)
			}

			got := map[expectedMessage]int{}
			for _, diagram := range bundle.Diagrams {
				for _, message := range diagram.Messages {
					if isInteractionDiagramMessage(message.Kind) {
						got[expectedMessage{caller: message.Caller, callee: message.Callee, kind: message.Kind}]++
					}
				}
			}
			for message, count := range test.want {
				if got[message] != count {
					t.Errorf("message %#v count = %d, want %d", message, got[message], count)
				}
			}
			if len(got) != len(test.want) {
				t.Errorf("messages = %#v, want %#v", got, test.want)
			}
		})
	}
}

//	{
//	  責務: [TestSequenceReferencesPointToActualRootDiagrams: 複数root系列の境界参照がnodeを含む実在図を指す]
//	}
func TestSequenceReferencesPointToActualRootDiagrams(t *testing.T) {
	module := sequenceTestModule(
		commonir.Entity{Kind: "function", Name: "caller_a", Calls: []string{"worker"}, CallSequence: []string{"worker"}},
		commonir.Entity{Kind: "function", Name: "caller_b", Calls: []string{"worker"}, CallSequence: []string{"worker"}},
		commonir.Entity{Kind: "function", Name: "worker"},
		commonir.Entity{Kind: "function", Name: "caller_c", Calls: []string{"logger"}, CallSequence: []string{"logger"}},
		commonir.Entity{Kind: "function", Name: "caller_d", Calls: []string{"logger"}, CallSequence: []string{"logger"}},
		commonir.Entity{Kind: "function", Name: "caller_e", Calls: []string{"logger"}, CallSequence: []string{"logger"}},
		commonir.Entity{Kind: "function", Name: "logger", Calls: []string{"caller_b"}, CallSequence: []string{"caller_b"}},
	)
	bundle, err := BuildSequenceDiagramBundle(module, 3, 8, nil)
	if err != nil {
		t.Fatalf("BuildSequenceDiagramBundle() error = %v", err)
	}
	diagrams := make(map[string]SequenceDiagram, len(bundle.Diagrams))
	for _, diagram := range bundle.Diagrams {
		diagrams[diagram.Name] = diagram
	}
	shared := diagrams[stableOutputName("shared", "logger", "sequence")]
	foundCallerBReference := false
	for _, reference := range shared.References {
		if reference.Node != "caller_b" {
			continue
		}
		foundCallerBReference = true
		target, exists := diagrams[reference.OutputName]
		if !exists {
			t.Fatalf("shared reference points to missing diagram %q", reference.OutputName)
		}
		if !containsString(target.Participants, "caller_b") {
			t.Errorf("shared reference target %q does not contain caller_b: %#v", target.Name, target.Participants)
		}
	}
	if !foundCallerBReference {
		t.Fatalf("shared diagram has no boundary reference to caller_b: %#v", shared.References)
	}
	for _, diagram := range bundle.Diagrams {
		for _, reference := range diagram.References {
			target, exists := diagrams[reference.OutputName]
			if !exists {
				t.Errorf("diagram %q references missing output %q", diagram.Name, reference.OutputName)
				continue
			}
			if !containsString(target.Participants, reference.Node) {
				t.Errorf("diagram %q reference node %q is absent from %q", diagram.Name, reference.Node, target.Name)
			}
		}
	}
}

//	{
//	  責務: [TestCallGraphBundleKeepsSharedBoundaryReferences: 通常call図とshared図の両方にshared境界を表示する]
//	}
func TestCallGraphBundleKeepsSharedBoundaryReferences(t *testing.T) {
	module := sequenceTestModule(
		commonir.Entity{Kind: "function", Name: "caller_a", Calls: []string{"logger"}, CallSequence: []string{"logger"}},
		commonir.Entity{Kind: "function", Name: "caller_b", Calls: []string{"logger"}, CallSequence: []string{"logger"}},
		commonir.Entity{Kind: "function", Name: "caller_c", Calls: []string{"logger"}, CallSequence: []string{"logger"}},
		commonir.Entity{Kind: "function", Name: "logger"},
	)
	bundle, err := BuildCallGraphBundle(module, 3, nil, nil)
	if err != nil {
		t.Fatalf("BuildCallGraphBundle() error = %v", err)
	}
	if len(bundle.Diagrams) != 4 {
		t.Fatalf("diagram count = %d, want three regular diagrams and one shared diagram", len(bundle.Diagrams))
	}
	for _, diagram := range bundle.Diagrams {
		if diagram.Name == stableOutputName("shared", "logger", "call_graph") {
			if len(diagram.Graph.Edges) != 3 || len(diagram.Graph.References) != 3 {
				t.Fatalf("shared diagram lost relations or regular diagram references: %#v", diagram.Graph)
			}
			continue
		}
		if len(diagram.Graph.Edges) != 1 || diagram.Graph.Edges[0].Callee != "logger" {
			t.Errorf("regular diagram lost shared relation: %#v", diagram.Graph)
		}
		if len(diagram.Graph.References) != 1 || diagram.Graph.References[0].OutputName != stableOutputName("shared", "logger", "call_graph") {
			t.Errorf("regular diagram has no shared reference: %#v", diagram.Graph.References)
		}
	}
}

//	{
//	  責務: [TestClassDiagramBundleKeepsSharedBoundaryReferences: class関係を通常図とshared図から辿れるようにする]
//	}
func TestClassDiagramBundleKeepsSharedBoundaryReferences(t *testing.T) {
	module := sequenceTestModule(
		commonir.Entity{Kind: "class", Name: "CallerA", Bases: []string{"Logger"}},
		commonir.Entity{Kind: "class", Name: "CallerB", Bases: []string{"Logger"}},
		commonir.Entity{Kind: "class", Name: "CallerC", Bases: []string{"Logger"}},
		commonir.Entity{Kind: "class", Name: "Logger"},
	)
	bundle, err := BuildClassDiagramBundle(module, 3, nil)
	if err != nil {
		t.Fatalf("BuildClassDiagramBundle() error = %v", err)
	}
	if len(bundle.Diagrams) != 4 {
		t.Fatalf("diagram count = %d, want three regular diagrams and one shared diagram", len(bundle.Diagrams))
	}
	sharedOutput := stableOutputName("shared", "Logger", "class")
	for _, diagram := range bundle.Diagrams {
		if diagram.Name == sharedOutput {
			if len(diagram.Relations) != 3 {
				t.Fatalf("shared diagram relations = %#v, want all three inheritance relations", diagram.Relations)
			}
			continue
		}
		if len(diagram.Relations) != 1 || diagram.Relations[0].Target != "Logger" {
			t.Errorf("regular diagram lost shared relation: %#v", diagram.Relations)
		}
		var sharedStub *ClassNode
		for index := range diagram.Nodes {
			if diagram.Nodes[index].Name == "Logger" {
				sharedStub = &diagram.Nodes[index]
			}
		}
		if sharedStub == nil || !sharedStub.External || sharedStub.ReferenceOutput != sharedOutput {
			t.Errorf("regular diagram has no shared stub reference: %#v", diagram.Nodes)
		}
	}
}

//	{
//	  責務: [TestClassDiagramOptionsDoNotCreateHiddenBoundaryReferences: 非表示にした関係から続き参照を作らない]
//	}
func TestClassDiagramOptionsDoNotCreateHiddenBoundaryReferences(t *testing.T) {
	module := sequenceTestModule(
		commonir.Entity{Kind: "class", Name: "CallerA", Bases: []string{"Logger"}},
		commonir.Entity{Kind: "class", Name: "CallerB", Bases: []string{"Logger"}},
		commonir.Entity{Kind: "class", Name: "CallerC", Bases: []string{"Logger"}},
		commonir.Entity{Kind: "class", Name: "Logger"},
	)
	options := DefaultClassDiagramOptions()
	options.IncludeInheritance = false
	bundle, err := BuildClassDiagramBundle(module, 3, &options)
	if err != nil {
		t.Fatalf("BuildClassDiagramBundle() error = %v", err)
	}
	if bundle.Statistics["shared_node_count"] != 1 {
		t.Errorf("shared node count = %v, want hidden semantic relation to remain in partition metadata", bundle.Statistics["shared_node_count"])
	}
	for _, diagram := range bundle.Diagrams {
		if len(diagram.Relations) != 0 {
			t.Errorf("hidden inheritance relations remain in %q: %#v", diagram.Name, diagram.Relations)
		}
		for _, node := range diagram.Nodes {
			if node.ReferenceOutput != "" {
				t.Errorf("hidden inheritance relation created a reference in %q: %#v", diagram.Name, node)
			}
		}
	}
}

//	{
//	  責務: [TestSequenceBundleMarksDepthTruncation: max-depthで省く内部callを再帰cycleと区別して記録する]
//	}
func TestSequenceBundleMarksDepthTruncation(t *testing.T) {
	module := sequenceTestModule(
		commonir.Entity{Kind: "function", Name: "entry", CallSequence: []string{"worker"}},
		commonir.Entity{Kind: "function", Name: "worker", CallSequence: []string{"leaf"}},
		commonir.Entity{Kind: "function", Name: "leaf"},
	)
	bundle, err := BuildSequenceDiagramBundle(module, 3, 1, nil)
	if err != nil {
		t.Fatalf("BuildSequenceDiagramBundle() error = %v", err)
	}

	var calls, truncations int
	for _, diagram := range bundle.Diagrams {
		for _, message := range diagram.Messages {
			switch message.Kind {
			case SequenceCallMessage:
				if message.Caller == "entry" && message.Callee == "worker" {
					calls++
				}
			case SequenceDepthTruncated:
				if message.Caller == "worker" && message.Callee == "leaf" {
					truncations++
				}
			case SequenceRecursiveCall:
				t.Errorf("depth truncation was mislabeled as recursion: %#v", message)
			}
		}
	}
	if calls != 1 || truncations != 1 {
		t.Fatalf("calls = %d, depth truncations = %d; want one of each", calls, truncations)
	}

	communication, err := BuildCommunicationDiagramBundle(module, 3, 1, true)
	if err != nil {
		t.Fatalf("BuildCommunicationDiagramBundle() error = %v", err)
	}
	var communicationTruncations int
	for _, diagram := range communication.Diagrams {
		for _, message := range diagram.Messages {
			if message.Kind == SequenceDepthTruncated && message.Source == "worker" && message.Target == "leaf" {
				communicationTruncations++
			}
		}
	}
	if communicationTruncations != 1 {
		t.Fatalf("communication depth truncations = %d; want one", communicationTruncations)
	}

	withReturns, err := BuildSequenceDiagramBundle(module, 3, 1, &SequenceDiagramOptions{ShowDuplicateCalls: true, ShowReturns: true})
	if err != nil {
		t.Fatalf("BuildSequenceDiagramBundle() with returns error = %v", err)
	}
	for _, diagram := range withReturns.Diagrams {
		for _, message := range diagram.Messages {
			if message.Kind == SequenceReturnMessage && ((message.Caller == "leaf" && message.Callee == "worker") || (message.Caller == "worker" && message.Callee == "leaf")) {
				t.Fatalf("depth-truncated call produced a return for an unexpanded target: %#v", message)
			}
		}
	}
}

//	{
//	  責務: [TestSequenceDepthContinuationsPreserveEveryInternalEdge: maxDepthを超える内部callを続き図と参照で保持する]
//	}
func TestSequenceDepthContinuationsPreserveEveryInternalEdge(t *testing.T) {
	module := sequenceTestModule(
		commonir.Entity{Kind: "function", Name: "alpha", CallSequence: []string{"beta"}},
		commonir.Entity{Kind: "function", Name: "beta", CallSequence: []string{"gamma"}},
		commonir.Entity{Kind: "function", Name: "gamma", CallSequence: []string{"delta"}},
		commonir.Entity{Kind: "function", Name: "delta", CallSequence: []string{"epsilon"}},
		commonir.Entity{Kind: "function", Name: "epsilon"},
	)
	sequences := analyzers.ResolveCallSequences(module)
	graph := analyzers.BuildSequenceRelationGraph(module, sequences)
	wantEdges := map[[2]string]struct{}{}
	for _, edge := range graph.Edges() {
		wantEdges[[2]string{edge.Caller, edge.Callee}] = struct{}{}
	}
	bundle, err := BuildSequenceDiagramBundle(module, 3, 1, nil)
	if err != nil {
		t.Fatalf("BuildSequenceDiagramBundle() error = %v", err)
	}
	diagramsByName := make(map[string]SequenceDiagram, len(bundle.Diagrams))
	gotEdges := map[[2]string]struct{}{}
	for _, diagram := range bundle.Diagrams {
		diagramsByName[diagram.Name] = diagram
		for _, message := range diagram.Messages {
			if isInteractionDiagramMessage(message.Kind) {
				gotEdges[[2]string{message.Caller, message.Callee}] = struct{}{}
			}
			if message.Kind != SequenceDepthTruncated {
				continue
			}
			if len(sequences[message.Callee]) == 0 {
				continue
			}
			foundReference := false
			for _, reference := range diagram.References {
				if reference.Node != message.Callee {
					continue
				}
				foundReference = true
				target, exists := diagramsByName[reference.OutputName]
				if !exists {
					target = sequenceDiagramByName(bundle.Diagrams, reference.OutputName)
				}
				if target.Name == "" || !containsString(target.Participants, message.Callee) {
					t.Errorf("truncated target %q has invalid continuation reference %#v", message.Callee, reference)
				}
			}
			if !foundReference {
				t.Errorf("diagram %q has no continuation reference for %q", diagram.Name, message.Callee)
			}
		}
	}
	if len(gotEdges) != len(wantEdges) {
		t.Fatalf("rendered call edges = %#v, want %#v", gotEdges, wantEdges)
	}
	for edge := range wantEdges {
		if _, exists := gotEdges[edge]; !exists {
			t.Errorf("internal call edge %q -> %q is missing from all diagrams", edge[0], edge[1])
		}
	}
}

//	{
//	  責務: [sequenceDiagramByName: bundle内からdiagram名で一つの図を取得する]
//	}
func sequenceDiagramByName(diagrams []SequenceDiagram, name string) SequenceDiagram {
	for _, diagram := range diagrams {
		if diagram.Name == name {
			return diagram
		}
	}
	return SequenceDiagram{}
}

//	{
//	  責務: [TestDepthTruncationPreservesSeriesExitCallsAndReturns: depth上限のseries外callとinternal returnを表示する]
//	}
func TestDepthTruncationPreservesSeriesExitCallsAndReturns(t *testing.T) {
	calls := []analyzers.ResolvedCall{{Target: "outside", Raw: "outside()"}, {Target: "library", Raw: "library()"}}
	messages := []SequenceMessage{}
	appendDepthTruncations(
		&messages,
		"caller",
		1,
		calls,
		map[string][]analyzers.ResolvedCall{"caller": calls, "outside": nil},
		map[string]struct{}{},
		map[string]struct{}{"caller": {}},
		map[string]struct{}{},
		SequenceDiagramOptions{ShowReturns: true},
	)

	if len(messages) != 3 {
		t.Fatalf("messages = %#v; want an outside call and return plus an external call", messages)
	}
	if messages[0].Kind != SequenceCallMessage || messages[0].Caller != "caller" || messages[0].Callee != "outside" {
		t.Errorf("internal series-exit call = %#v", messages[0])
	}
	if messages[1].Kind != SequenceReturnMessage || messages[1].Caller != "outside" || messages[1].Callee != "caller" {
		t.Errorf("internal series-exit return = %#v", messages[1])
	}
	if messages[2].Kind != SequenceCallMessage || messages[2].Callee != "library" {
		t.Errorf("external call = %#v", messages[2])
	}
}

//	{
//	  責務: [TestAppendSequenceCallDeduplicatesRecursionByKind: 重複非表示時もnormal callとcycle-closing callを別のmessageとして保つ]
//	}
func TestAppendSequenceCallDeduplicatesRecursionByKind(t *testing.T) {
	messages := []SequenceMessage{}
	emitted := map[string]struct{}{}
	options := SequenceDiagramOptions{ShowDuplicateCalls: false}
	if !appendSequenceCall(&messages, emitted, "beta", "alpha", "visit", 0, SequenceCallMessage, options) {
		t.Fatal("normal call was not appended")
	}
	if !appendSequenceCall(&messages, emitted, "beta", "alpha", "visit", 0, SequenceRecursiveCall, options) {
		t.Fatal("recursive call was hidden by the matching normal call")
	}
	if appendSequenceCall(&messages, emitted, "beta", "alpha", "visit", 0, SequenceRecursiveCall, options) {
		t.Fatal("duplicate recursive call was appended while duplicates are disabled")
	}
	if !appendSequenceCall(&messages, emitted, "beta", "alpha", "visit", 0, SequenceDepthTruncated, options) {
		t.Fatal("depth-truncation message was hidden by another message kind")
	}
	if appendSequenceCall(&messages, emitted, "beta", "alpha", "visit", 0, SequenceDepthTruncated, options) {
		t.Fatal("duplicate depth-truncation message was appended while duplicates are disabled")
	}
	if !appendSequenceCall(&messages, emitted, "beta", "alpha", "visit another path", 0, SequenceDepthTruncated, options) {
		t.Fatal("depth-truncation message for a distinct call expression was hidden")
	}
}

func sequenceTestModule(entities ...commonir.Entity) commonir.Module {
	return commonir.Module{Language: "go", Entities: entities}
}
