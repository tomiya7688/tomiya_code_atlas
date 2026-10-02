package pythonadapter

import (
	"os"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

func TestParseExtractsDeclarationsSignaturesDocsAndCalls(t *testing.T) {
	source := `"""module docs"""
class Worker(Base):
    """worker docs"""
    async def run(self, value: int, flag: bool = True) -> str:
        """run docs"""
        return self.format(value)
    def format(self, value: int) -> str:
        return str(value)
`
	got := Parse(source)
	if err := Validate(got); err != nil {
		t.Fatal(err)
	}
	if got.ModuleDocstring == nil || *got.ModuleDocstring != "module docs" {
		t.Fatalf("module docstring: %#v", got.ModuleDocstring)
	}
	if len(got.Entities) != 3 {
		t.Fatalf("expected class and two methods, got %#v", got.Entities)
	}
	worker, run, format := got.Entities[0], got.Entities[1], got.Entities[2]
	if worker.Kind != "class" || worker.Bases[0] != "Base" || *worker.Docstring != "worker docs" || worker.Line != 2 {
		t.Fatalf("class facts: %#v", worker)
	}
	if run.Kind != "method" || !run.IsAsync || run.Line != 4 || *run.Docstring != "run docs" || len(run.Parameters) != 3 || run.Parameters[2] != "flag" || *run.ReturnType != "str" {
		t.Fatalf("function facts: %#v", run)
	}
	if len(run.CallSequence) != 1 || run.CallSequence[0] != "self.format" || len(run.ResolvedCalls) != 1 || run.ResolvedCalls[0] != "Worker.format" || format.Line != 7 {
		t.Fatalf("call and location facts: run=%#v format=%#v", run, format)
	}
}

func TestParseKeepsMultilineDocstringsAsOneFact(t *testing.T) {
	got := Parse("\"\"\"module\ndescription\n\"\"\"\ndef run():\n    \"\"\"function\ndescription\n    \"\"\"\n    return 1\n")
	if got.ModuleDocstring == nil || *got.ModuleDocstring != "module\ndescription" {
		t.Fatalf("module docstring: %#v", got.ModuleDocstring)
	}
	if len(got.Entities) != 1 || got.Entities[0].Docstring == nil || *got.Entities[0].Docstring != "function\ndescription" || got.Entities[0].EndLine != 8 {
		t.Fatalf("function facts: %#v", got.Entities)
	}
}

func TestAmbiguousCallsStayAsUnresolvedSourceFacts(t *testing.T) {
	got := Parse("class A:\n    def target(self):\n        pass\nclass B:\n    def target(self):\n        pass\ndef run():\n    target()\n")
	var run *commonir.Entity
	for i := range got.Entities {
		if got.Entities[i].Name == "run" {
			run = &got.Entities[i]
		}
	}
	if run == nil || len(run.Calls) != 1 || run.Calls[0] != "target" || len(run.ResolvedCalls) != 0 {
		t.Fatalf("ambiguous call should remain unresolved: %#v", run)
	}
}

func TestBackendConformanceFixtureParsesAsCommonIR(t *testing.T) {
	data, err := os.ReadFile("../../../tests/fixtures/backend_conformance/python.py")
	if err != nil {
		t.Fatal(err)
	}
	got := Parse(string(data))
	if err := pythonIRValid(got); err != nil {
		t.Fatalf("adapter output rejected by Common IR codec: %v", err)
	}
	if len(got.Entities) == 0 {
		t.Fatal("fixture produced no declarations")
	}
	var worker, run, asyncProbe, nestedProbe, nested *commonir.Entity
	for i := range got.Entities {
		e := &got.Entities[i]
		switch e.Name {
		case "Worker":
			worker = e
		case "run":
			run = e
		case "async_probe":
			asyncProbe = e
		case "nested_probe":
			nestedProbe = e
		case "nested":
			nested = e
		}
	}
	if worker == nil || len(worker.Bases) != 2 || worker.Bases[0] != "BaseWorker" {
		t.Fatalf("inheritance facts missing: %#v", worker)
	}
	if run == nil || len(run.CallSequence) != 2 || len(run.Calls) != 1 || len(run.ResolvedCalls) != 1 || run.ResolvedCalls[0] != "Worker.helper" {
		t.Fatalf("call facts and confidence were not preserved: %#v", run)
	}
	if asyncProbe == nil || !asyncProbe.IsAsync {
		t.Fatalf("async declaration was not recognized: %#v", asyncProbe)
	}
	if nestedProbe == nil || nested == nil || nested.Kind != "function" || nested.Parent == nil || *nested.Parent != "Worker.nested_probe" {
		t.Fatalf("nested function scope was not preserved: probe=%#v nested=%#v", nestedProbe, nested)
	}
}

func pythonIRValid(payload commonir.Payload) error { _, err := commonir.Encode(payload); return err }
