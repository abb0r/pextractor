package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ncruces/zenity"
)

type exportDoc struct {
	Server  string
	Library string
	Kind    string
	Lines   []string
}

func saveExport(doc exportDoc) (string, error) {
	path, err := zenity.SelectFileSave(
		zenity.Title("Als TXT exportieren"),
		zenity.Filename(exportFileName(doc.Library)),
		zenity.ConfirmOverwrite(),
		zenity.FileFilters{
			{Name: "Textdatei", Patterns: []string{"*.txt"}},
		},
	)
	if err != nil {
		if errors.Is(err, zenity.ErrCanceled) {
			return "", nil
		}
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(path), ".txt") {
		path += ".txt"
	}
	if err := writeExport(path, doc); err != nil {
		return "", err
	}
	return path, nil
}

func writeExport(path string, doc exportDoc) error {
	var b strings.Builder
	b.WriteString("Pextractor\r\n")
	fmt.Fprintf(&b, "Server: %s\r\n", doc.Server)
	fmt.Fprintf(&b, "Bibliothek: %s\r\n", doc.Library)
	fmt.Fprintf(&b, "Art: %s\r\n", doc.Kind)
	fmt.Fprintf(&b, "Anzahl: %d\r\n\r\n", len(doc.Lines))
	for _, line := range doc.Lines {
		line = strings.ReplaceAll(line, "\r\n", " ")
		line = strings.ReplaceAll(line, "\n", " ")
		line = strings.ReplaceAll(line, "\r", " ")
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(b.String())...)
	return os.WriteFile(path, data, 0o644)
}
