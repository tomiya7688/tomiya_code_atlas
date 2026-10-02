// Package conformance validates the versioned, renderer-neutral Common IR golden.
package conformance

import "github.com/tomiya7688/tomiya_code_atlas/internal/commonir"

// ValidateSerializedFixture checks the typed Common IR and logical-output fields.
// Unknown fields are intentionally accepted so optional additions remain compatible.
func ValidateSerializedFixture(data []byte) error {
	_, err := commonir.DecodeFixture(data)
	return err
}
