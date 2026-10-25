package process

import "testing"

func TestBoundedCaptureKeepsOnlyConfiguredPrefix(t *testing.T) {
	capture := newBoundedCapture(4)

	written, err := capture.Write([]byte("abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if written != 6 {
		t.Fatalf("Write reported %d bytes, want 6", written)
	}
	if got := string(capture.Bytes()); got != "abcd" {
		t.Fatalf("captured %q, want %q", got, "abcd")
	}
	if !capture.truncated {
		t.Fatal("capture must report truncation")
	}

	written, err = capture.Write([]byte("gh"))
	if err != nil {
		t.Fatal(err)
	}
	if written != 2 {
		t.Fatalf("second Write reported %d bytes, want 2", written)
	}
	if got := string(capture.Bytes()); got != "abcd" {
		t.Fatalf("capture grew after reaching limit: %q", got)
	}
}
