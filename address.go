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
		return "", "", errors.New("Please enter the server address.")
	}
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	u, err := url.Parse(server)
	if err != nil || u.Host == "" {
		return "", "", errors.New("The server address is invalid.")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", errors.New("Only http and https are supported.")
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
		return "", "", errors.New("Please enter the Plex token.")
	}
	return u.String(), token, nil
}

func exportFileName(library string) string {
	library = strings.TrimSpace(library)
	if library == "" {
		library = "pExtractor"
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
		name = "pExtractor"
	}
	return name + ".txt"
}

func typeLabel(kind string) string {
	switch kind {
	case "movie":
		return "Movies"
	case "show":
		return "TV shows"
	case "artist":
		return "Music"
	case "photo":
		return "Photos"
	default:
		if kind == "" {
			return "Unknown"
		}
		return kind
	}
}

func countLabel(n int, kind string) string {
	switch kind {
	case "movie":
		if n == 1 {
			return "1 movie"
		}
		return plural(n, "movies")
	case "show":
		if n == 1 {
			return "1 series"
		}
		return plural(n, "series")
	case "artist":
		if n == 1 {
			return "1 artist"
		}
		return plural(n, "artists")
	default:
		if n == 1 {
			return "1 entry"
		}
		return plural(n, "entries")
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
