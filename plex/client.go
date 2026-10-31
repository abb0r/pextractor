// Package plex reads library titles from a Plex Media Server.
package plex

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const pageSize = 400

// Library is one Plex library section.
type Library struct {
	Key   string
	Title string
	Type  string
}

// Supported reports whether titles or artists can be read from this library.
func (l Library) Supported() bool {
	switch l.Type {
	case "movie", "show", "artist":
		return true
	default:
		return false
	}
}

// Client talks to one Plex Media Server.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient builds a client for baseURL. Token is sent as X-Plex-Token.
// insecureTLS accepts servers with a self-signed certificate.
func NewClient(baseURL, token string, insecureTLS bool) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("Bitte die Server-Adresse angeben.")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("Bitte den Plex-Token angeben.")
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("Die Server-Adresse ist ungültig.")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Transport: transport},
	}, nil
}

// Libraries returns every library section on the server, sorted by title.
func (c *Client) Libraries(ctx context.Context) ([]Library, error) {
	var payload container
	if err := c.get(ctx, "/library/sections", nil, &payload); err != nil {
		return nil, err
	}
	libs := make([]Library, 0, len(payload.MediaContainer.Directory))
	for _, dir := range payload.MediaContainer.Directory {
		title := strings.TrimSpace(dir.Title)
		key := strings.TrimSpace(string(dir.Key))
		kind := strings.TrimSpace(dir.Type)
		if title == "" || key == "" {
			continue
		}
		libs = append(libs, Library{Key: key, Title: title, Type: kind})
	}
	sortLibraries(libs)
	return libs, nil
}

// Entries reads movie titles, series titles, or artist names from lib.
// progress may be nil. loaded and total are item counts; total is 0 until the server reports it.
func (c *Client) Entries(ctx context.Context, lib Library, progress func(loaded, total int)) ([]string, error) {
	typeCode, err := typeCode(lib.Type)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var titles []string
	start := 0
	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("type", typeCode)
		q.Set("X-Plex-Container-Start", fmt.Sprintf("%d", start))
		q.Set("X-Plex-Container-Size", fmt.Sprintf("%d", pageSize))
		var payload container
		if err := c.get(ctx, "/library/sections/"+url.PathEscape(lib.Key)+"/all", q, &payload); err != nil {
			return nil, err
		}
		mc := payload.MediaContainer
		if mc.TotalSize > 0 {
			total = mc.TotalSize
		}
		if len(mc.Metadata) == 0 {
			break
		}
		added := 0
		for _, item := range mc.Metadata {
			if item.Type != "" && item.Type != lib.Type {
				continue
			}
			key := strings.TrimSpace(string(item.RatingKey))
			if key != "" {
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
			}
			title := strings.TrimSpace(item.Title)
			if title == "" {
				title = strings.TrimSpace(item.OriginalTitle)
			}
			if title == "" {
				continue
			}
			titles = append(titles, title)
			added++
		}
		if progress != nil {
			progress(len(titles), total)
		}
		start += len(mc.Metadata)
		if added == 0 || len(mc.Metadata) < pageSize || (total > 0 && start >= total) {
			break
		}
		if len(titles) > 500000 {
			return nil, errors.New("Die Bibliothek enthält zu viele Einträge.")
		}
	}
	sortStrings(titles)
	return titles, nil
}

func typeCode(kind string) (string, error) {
	switch kind {
	case "movie":
		return "1", nil
	case "show":
		return "2", nil
	case "artist":
		return "8", nil
	default:
		return "", fmt.Errorf("Bibliothekstyp %q wird nicht unterstützt", kind)
	}
}

type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(strings.TrimSpace(string(b)))
	return nil
}

type container struct {
	MediaContainer struct {
		Size      int `json:"size"`
		TotalSize int `json:"totalSize"`
		Directory []struct {
			Key   flexString `json:"key"`
			Title string     `json:"title"`
			Type  string     `json:"type"`
		} `json:"Directory"`
		Metadata []struct {
			RatingKey     flexString `json:"ratingKey"`
			Title         string     `json:"title"`
			OriginalTitle string     `json:"originalTitle"`
			Type          string     `json:"type"`
		} `json:"Metadata"`
	} `json:"MediaContainer"`
}

func (c *Client) get(ctx context.Context, path string, query url.Values, dest any) error {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return errors.New("Die Server-Adresse ist ungültig.")
	}
	full := *base
	full.Path = strings.TrimRight(base.Path, "/") + path
	full.RawQuery = ""
	if len(query) > 0 {
		full.RawQuery = query.Encode()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, full.String(), nil)
	if err != nil {
		return friendly(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("X-Plex-Product", "Pextractor")
	req.Header.Set("X-Plex-Version", "1.0.0")
	req.Header.Set("X-Plex-Client-Identifier", "pextractor")
	resp, err := c.http.Do(req)
	if err != nil {
		return friendly(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return friendly(err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.New("Zugriff verweigert. Token und Berechtigung prüfen.")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Unerwartete Antwort vom Server (HTTP %d).", resp.StatusCode)
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' {
		return errors.New("Der Server hat kein JSON geliefert. Adresse und Port 32400 prüfen.")
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return errors.New("Die Antwort des Servers konnte nicht gelesen werden.")
	}
	return nil
}

func friendly(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("Zeitüberschreitung beim Abruf vom Plex-Server.")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errors.New("Zeitüberschreitung beim Abruf vom Plex-Server.")
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"):
		return errors.New("Verbindung abgelehnt. Läuft der Plex-Server und stimmt die Adresse?")
	case strings.Contains(msg, "no such host"):
		return errors.New("Server nicht gefunden. Adresse prüfen.")
	case strings.Contains(msg, "certificate") || strings.Contains(msg, "x509"):
		return errors.New("Zertifikatsfehler. Für selbstsignierte Zertifikate die Option unten aktivieren.")
	default:
		return fmt.Errorf("Verbindung fehlgeschlagen: %v", err)
	}
}

func sortLibraries(items []Library) {
	coll := germanCollator()
	sort.SliceStable(items, func(i, j int) bool {
		return coll.CompareString(items[i].Title, items[j].Title) < 0
	})
}

func sortStrings(items []string) {
	coll := germanCollator()
	sort.SliceStable(items, func(i, j int) bool {
		return coll.CompareString(items[i], items[j]) < 0
	})
}

func germanCollator() *collate.Collator {
	return collate.New(language.German, collate.IgnoreCase, collate.Loose)
}
