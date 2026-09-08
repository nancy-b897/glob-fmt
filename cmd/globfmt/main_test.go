package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"[^abc]*.log", "src/{a,b}.go"}, strings.NewReader(""), &stdout, &stderr)
	if status != 0 {
		t.Fatalf("status = %d, want 0; stderr = %q", status, stderr.String())
	}
	want := "[!abc]*.log\nsrc/{a,b}.go\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunInvalidPatternContinues(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"a**b", "*.go"}, strings.NewReader(""), &stdout, &stderr)
	if status != 1 {
		t.Fatalf("status = %d, want 1", status)
	}
	if stdout.String() != "*.go\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "*.go\n")
	}
	if !strings.Contains(stderr.String(), "a**b") {
		t.Fatalf("stderr = %q, want it to mention the bad pattern", stderr.String())
	}
}

func TestRunReadsStdinWhenNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	in := "*.go\n\nsrc/{a,b}.go\n"
	status := run(nil, strings.NewReader(in), &stdout, &stderr)
	if status != 0 {
		t.Fatalf("status = %d, want 0; stderr = %q", status, stderr.String())
	}
	want := "*.go\nsrc/{a,b}.go\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}
