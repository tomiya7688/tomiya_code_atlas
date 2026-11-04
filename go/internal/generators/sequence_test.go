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
