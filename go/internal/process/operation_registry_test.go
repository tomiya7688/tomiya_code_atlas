package process

import "testing"

//	{
//	  責務: [TestSupportedOperationIDsReturnsStableCopy: CLI operationの表示順と内部registryの保護を確認する]
//	}
func TestSupportedOperationIDsReturnsStableCopy(t *testing.T) {
	expected := []string{"call-graph", "class-diagram", "comments", "static-spec", "design-quality"}
	actual := SupportedOperationIDs()
	if len(actual) != len(expected) {
		t.Fatalf("SupportedOperationIDs = %#v, want %#v", actual, expected)
	}
	for index, operation := range expected {
		if actual[index] != operation || !IsSupportedOperation(operation) {
			t.Errorf("operation at index %d = %q, want supported %q", index, actual[index], operation)
		}
	}
	actual[0] = "changed"
	if !IsSupportedOperation("call-graph") || IsSupportedOperation("changed") {
		t.Fatal("changing the returned operation slice changed the registry")
	}
}
