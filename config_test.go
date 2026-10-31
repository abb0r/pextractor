package main

import (
	"path/filepath"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pExtractor", "config.json")
	want := connectionConfig{Server: "http://127.0.0.1:32400", Token: "secret", Insecure: true}
	if err := writeConfig(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := readConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %#v", got)
	}
}
