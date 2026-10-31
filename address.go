package main

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

func normalize(server, token string) (string, string, error) {
	server = strings.TrimSpace(server)
	token = strings.TrimSpace(token)
	if server == "" {
		return "", "", errors.New("Bitte die Server-Adresse angeben.")
	}
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	u, err := url.Parse(server)
	if err != nil || u.Host == "" {
		return "", "", errors.New("Die Server-Adresse ist ungültig.")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", errors.New("Nur http und https werden unterstützt.")
	}
	if embedded := strings.TrimSpace(u.Query().Get("X-Plex-Token")); embedded != "" && token == "" {
		token = embedded
	}
	u.RawQuery = ""
	u.Fragment = ""
	path := strings.TrimRight(u.Path, "/")
	if path == "/web" || strings.HasPrefix(path, "/web/") {
		path = ""
	}
	u.Path = path
	if token == "" {
		return "", "", errors.New("Bitte den Plex-Token angeben.")
	}
	return u.String(), token, nil
}

func exportFileName(library string) string {
	library = strings.TrimSpace(library)
	if library == "" {
		library = "pextractor"
	}
	var b strings.Builder
	for _, r := range library {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			b.WriteByte('-')
		default:
			if unicode.IsControl(r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	name := strings.Trim(b.String(), " .")
	if name == "" {
		name = "pextractor"
	}
	return name + ".txt"
}

func typeLabel(kind string) string {
	switch kind {
	case "movie":
		return "Filme"
	case "show":
		return "Serien"
	case "artist":
		return "Musik"
	case "photo":
		return "Fotos"
	default:
		if kind == "" {
			return "Unbekannt"
		}
		return kind
	}
}

func countLabel(n int, kind string) string {
	switch kind {
	case "movie":
		if n == 1 {
			return "1 Film"
		}
		return plural(n, "Filme")
	case "show":
		if n == 1 {
			return "1 Serie"
		}
		return plural(n, "Serien")
	case "artist":
		if n == 1 {
			return "1 Künstler"
		}
		return plural(n, "Künstler")
	default:
		if n == 1 {
			return "1 Eintrag"
		}
		return plural(n, "Einträge")
	}
}

func plural(n int, word string) string {
	return itoa(n) + " " + word
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
