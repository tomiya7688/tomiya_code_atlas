package generators

import (
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [TestBuildStaticSpecificationCollectsResolvedRelationships: 解決可能な関係を集め曖昧な呼び出しを推測しないことを検証する]
//	}
func TestBuildStaticSpecificationCollectsResolvedRelationships(t *testing.T) {
	module := commonir.Module{
		Language: "python",
		Entities: []commonir.Entity{
			{Kind: "class", Name: "Worker"},
			{Kind: "method", Name: "run", Parent: staticParentPointer("Worker"), Calls: []string{"self.format", "format", "external"}},
			{Kind: "method", Name: "format", Parent: staticParentPointer("Worker")},
			{Kind: "function", Name: "format"},
		},
		Imports: []string{}, Diagnostics: []commonir.Diagnostic{},
	}

	document := BuildStaticSpecification(module, "worker.py")
	if document.SourceName != "worker.py" || len(document.Declarations) != len(module.Entities) {
		t.Fatalf("unexpected generated document: %#v", document)
	}
	if got, want := document.Declarations[1].ResolvedCalls, []string{"Worker.format"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved calls = %v, want %v", got, want)
	}
	if got, want := document.Declarations[2].CalledBy, []string{"Worker.run"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("called by = %v, want %v", got, want)
	}
}

func staticParentPointer(value string) *string { return &value }
