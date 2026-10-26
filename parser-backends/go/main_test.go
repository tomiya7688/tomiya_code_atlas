package main

import (
	"encoding/json"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

func TestHelperIRUsesSharedCommonModel(t *testing.T) {
	result := parse(request{
		ContractVersion: "1",
		RequestID:       "common-ir-smoke",
		Operation:       "parse",
		Language:        "go",
		Path:            "sample.go",
		Source:          "package sample\nfunc Run() {}\n",
	})
	if !result.OK || result.IR == nil {
		t.Fatalf("helper parse failed: %#v", result)
	}
	wire, err := json.Marshal(result.IR)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := commonir.Decode(wire)
	if err != nil {
		t.Fatalf("shared model rejected helper payload: %v", err)
	}
	if decoded.Language != "go" || len(decoded.Entities) != 2 || decoded.Entities[1].Name != "Run" {
		t.Fatalf("unexpected shared model: %#v", decoded)
	}
}
