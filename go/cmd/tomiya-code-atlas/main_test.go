package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run returned %d; want 0", code)
	}
	if !strings.Contains(stdout.String(), "使用方法:") {
		t.Fatalf("help output is missing usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote to stderr: %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run returned %d; want 0", code)
	}
	if !strings.Contains(stdout.String(), "Tomiya Code Atlas (Go)") {
		t.Fatalf("version output is missing product name: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("version wrote to stderr: %q", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"class-diagram"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run returned %d; want 2", code)
	}
	if !strings.Contains(stderr.String(), "[エラー]") {
		t.Fatalf("error output is not Japanese: %q", stderr.String())
	}
}
