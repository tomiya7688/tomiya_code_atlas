package generators

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

func TestCommentCandidatesMatchGolden(t *testing.T) {
	fixtureBytes, err := os.ReadFile("../../../tests/fixtures/generators/comment_candidates_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Module   commonir.Module           `json:"module"`
		Expected []models.CommentCandidate `json:"expected_candidates"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	actual, err := GenerateCommentCandidates(fixture.Module)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, fixture.Expected) {
		t.Fatalf("comment candidates differ from golden\nactual:   %#v\nexpected: %#v", actual, fixture.Expected)
	}
}

func TestCommentCandidateRejectsInvalidSourcePosition(t *testing.T) {
	_, err := GenerateCommentCandidates(commonir.Module{Entities: []commonir.Entity{{Kind: "function", Name: "load", SourceLocation: commonir.SourceLocation{Line: 0}}}})
	if err == nil {
		t.Fatal("expected invalid line error")
	}
	_, err = GenerateCommentCandidates(commonir.Module{Entities: []commonir.Entity{{Kind: "function", Name: "load", SourceLocation: commonir.SourceLocation{Line: 1}, Indent: -1}}})
	if err == nil {
		t.Fatal("expected negative indent error")
	}
}
