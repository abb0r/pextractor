package main

import "testing"

func TestNormalize(t *testing.T) {
	base, token, err := normalize("192.168.1.20:32400/web", "")
	if err == nil || token != "" || base != "" {
		t.Fatalf("missing token: %q %q %v", base, token, err)
	}
	base, token, err = normalize("http://192.168.1.20:32400/web/index.html?X-Plex-Token=abc", "")
	if err != nil {
		t.Fatal(err)
	}
	if base != "http://192.168.1.20:32400" || token != "abc" {
		t.Fatalf("got %q %q", base, token)
	}
	base, token, err = normalize("https://plex.example:32400/", " explicit ")
	if err != nil {
		t.Fatal(err)
	}
	if base != "https://plex.example:32400" || token != "explicit" {
		t.Fatalf("got %q %q", base, token)
	}
}

func TestExportFileName(t *testing.T) {
	if got := exportFileName(`Filme: "A/B"`); got != "Filme- -A-B-.txt" {
		t.Fatalf("got %q", got)
	}
	if got := exportFileName("  "); got != "pExtractor.txt" {
		t.Fatalf("blank = %q", got)
	}
}

func TestCountLabel(t *testing.T) {
	if countLabel(1, "show") != "1 series" || countLabel(2, "artist") != "2 artists" {
		t.Fatalf("%q %q", countLabel(1, "show"), countLabel(2, "artist"))
	}
}
