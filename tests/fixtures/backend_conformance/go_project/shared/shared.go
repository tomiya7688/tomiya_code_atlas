package shared

// Result is shared by two Go packages.
type Result struct {
	Value string
}

// Normalize returns the canonical value.
func Normalize(input string) Result {
	return Result{Value: input}
}

