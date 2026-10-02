package main

import (
	"encoding/json"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

func TestHandleReturnsContractV1CommonIR(t *testing.T) {
	got := handle(request{ContractVersion: "1", RequestID: "unit-1", Operation: "parse", Language: "python", Source: "def run():\n    pass\n"})
	if !got.OK || got.RequestID != "unit-1" || got.IR == nil {
		t.Fatalf("unexpected helper response: %#v", got)
	}
	wire, err := json.Marshal(got.IR)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := commonir.Decode(wire)
	if err != nil {
		t.Fatalf("helper payload rejected by shared codec: %v", err)
	}
	if decoded.Language != "python" || len(decoded.Entities) != 1 || decoded.Entities[0].Name != "run" {
		t.Fatalf("unexpected IR: %#v", decoded)
	}
}

func TestHandleRejectsUnsupportedRequest(t *testing.T) {
	got := handle(request{ContractVersion: "9", RequestID: "bad", Operation: "parse", Language: "python"})
	if got.OK || got.Error == nil || got.Error.Kind != "protocol_error" || got.RequestID != "bad" {
		t.Fatalf("expected normalized protocol error, got %#v", got)
	}
}
