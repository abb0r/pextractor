package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	err := writeExport(path, []string{"The Bridge", "Still one\nmore line"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "The Bridge\nStill one more line\n"
	if string(got) != want {
		t.Fatalf("got %q", got)
	}
}
