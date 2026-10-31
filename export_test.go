package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	err := writeExport(path, exportDoc{
		Server:  "http://127.0.0.1:32400",
		Library: "Shows",
		Kind:    "TV shows",
		Lines:   []string{"The Bridge", "Still one\nmore line"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("missing utf-8 bom")
	}
	text := string(got[3:])
	for _, part := range []string{
		"Library: Shows\r\n",
		"Type: TV shows\r\n",
		"Count: 2\r\n",
		"The Bridge\r\n",
		"Still one more line\r\n",
	} {
		if !bytes.Contains(got, []byte(part)) {
			t.Fatalf("missing %q in %q", part, text)
		}
	}
}
