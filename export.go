package main

import (
	"errors"
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
		zenity.Title("Export as TXT"),
		zenity.Filename(exportFileName(doc.Library)),
		zenity.ConfirmOverwrite(),
		zenity.FileFilters{
			{Name: "Text file", Patterns: []string{"*.txt"}},
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
	if err := writeExport(path, doc.Lines); err != nil {
		return "", err
	}
	return path, nil
}

func writeExport(path string, lines []string) error {
	var b strings.Builder
	for _, line := range lines {
		line = strings.ReplaceAll(line, "\r\n", " ")
		line = strings.ReplaceAll(line, "\n", " ")
		line = strings.ReplaceAll(line, "\r", " ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
