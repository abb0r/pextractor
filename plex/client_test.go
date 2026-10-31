package plex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestLibrariesAndMovieTitles(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Plex-Token")
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/library/sections":
			_, _ = w.Write([]byte(`{
				"MediaContainer": {
					"size": 3,
					"Directory": [
						{"key": "2", "type": "show", "title": "Serien"},
						{"key": 1, "type": "movie", "title": "Ältere Filme"},
						{"key": "3", "type": "artist", "title": "Musik"}
					]
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/library/sections/1/all"):
			if r.URL.Query().Get("type") != "1" {
				t.Errorf("type = %s", r.URL.Query().Get("type"))
			}
			start := r.URL.Query().Get("X-Plex-Container-Start")
			if start == "0" {
				_, _ = w.Write([]byte(`{
					"MediaContainer": {
						"size": 2,
						"totalSize": 3,
						"Metadata": [
							{"ratingKey": "a", "type": "movie", "title": "Zebra"},
							{"ratingKey": "b", "type": "movie", "title": "äpfel"}
						]
					}
				}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"MediaContainer": {
					"size": 1,
					"totalSize": 3,
					"Metadata": [
						{"ratingKey": "c", "type": "movie", "title": "Banane"},
						{"ratingKey": "a", "type": "movie", "title": "Zebra"}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "secret-token", false)
	if err != nil {
		t.Fatal(err)
	}
	libs, err := c.Libraries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotToken != "secret-token" {
		t.Fatalf("token header = %q", gotToken)
	}
	if len(libs) != 3 {
		t.Fatalf("libraries = %#v", libs)
	}
	if libs[0].Title != "Ältere Filme" || libs[0].Key != "1" || libs[0].Type != "movie" {
		t.Fatalf("first library = %#v", libs[0])
	}

	titles, err := c.Entries(context.Background(), libs[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(titles) != 2 || titles[0] != "äpfel" || titles[1] != "Zebra" {
		t.Fatalf("titles = %#v", titles)
	}
	if !libs[2].Supported() || (Library{Type: "photo"}).Supported() {
		t.Fatal("supported flag mismatch")
	}
}

func TestPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("X-Plex-Container-Start")
		w.Header().Set("Content-Type", "application/json")
		if start == "0" {
			items := make([]string, 0, pageSize)
			for i := 0; i < pageSize; i++ {
				items = append(items, `{"ratingKey":"`+itoa(i)+`","type":"artist","title":"Künstler `+itoa(pageSize-i)+`"}`)
			}
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":` + itoa(pageSize) + `,"totalSize":` + itoa(pageSize+1) + `,"Metadata":[` + strings.Join(items, ",") + `]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"MediaContainer":{"size":1,"totalSize":` + itoa(pageSize+1) + `,"Metadata":[{"ratingKey":"last","type":"artist","title":"A Anfang"}]}}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "t", false)
	if err != nil {
		t.Fatal(err)
	}
	var progress []int
	titles, err := c.Entries(context.Background(), Library{Key: "9", Title: "Musik", Type: "artist"}, func(loaded, total int) {
		progress = append(progress, loaded)
		if total != pageSize+1 {
			t.Errorf("total = %d", total)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(titles) != pageSize+1 {
		t.Fatalf("len = %d", len(titles))
	}
	if titles[0] != "A Anfang" {
		t.Fatalf("first = %q", titles[0])
	}
	if len(progress) != 2 || progress[0] != pageSize || progress[1] != pageSize+1 {
		t.Fatalf("progress = %v", progress)
	}
}

func TestUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c, err := NewClient(srv.URL, "bad", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Libraries(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Access denied") {
		t.Fatalf("err = %v", err)
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
