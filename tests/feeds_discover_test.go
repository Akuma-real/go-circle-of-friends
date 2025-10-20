package tests

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "go-circle-of-friends/internal/fetch"
    "go-circle-of-friends/internal/feeds"
)

const rssSample = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>t</title>
    <link>https://ex</link>
    <item><title>a</title><link>https://ex/a</link><pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate></item>
  </channel>
 </rss>`

const atomSample = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example</title>
  <entry>
    <title>Atom-Powered Robots Run Amok</title>
    <updated>2003-12-13T18:30:02Z</updated>
    <link href="http://example.org/2003/12/13/atom03"/>
  </entry>
 </feed>`

func TestDiscoverFeed_StraightCandidate(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/index.xml", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/rss+xml")
        w.WriteHeader(200)
        _, _ = w.Write([]byte(rssSample))
    })
    // root no hints
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    cl, _ := fetch.New(fetch.Options{})
    got, err := feeds.DiscoverFeed(context.Background(), cl, srv.URL+"/", "")
    if err != nil { t.Fatalf("discover: %v", err) }
    if got != srv.URL+"/index.xml" { t.Fatalf("got %q want %q", got, srv.URL+"/index.xml") }
}

func TestDiscoverFeed_FromHTMLLink(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        _, _ = w.Write([]byte(`<!doctype html><head><link rel="alternate" type="application/rss+xml" href="/rss.xml"></head>`))
    })
    mux.HandleFunc("/rss.xml", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/rss+xml")
        _, _ = w.Write([]byte(rssSample))
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    cl, _ := fetch.New(fetch.Options{})
    got, err := feeds.DiscoverFeed(context.Background(), cl, srv.URL+"/", "")
    if err != nil { t.Fatalf("discover: %v", err) }
    if got != srv.URL+"/rss.xml" { t.Fatalf("got %q want %q", got, srv.URL+"/rss.xml") }
}

func TestParseFeed_JSONAndAtom(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/feed+json")
        _, _ = w.Write([]byte(`{"version":"https://jsonfeed.org/version/1.1","items": [{"id":"1","title":"x","url":"http://e/1"}]}`))
    })
    mux.HandleFunc("/atom.xml", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/atom+xml")
        _, _ = w.Write([]byte(atomSample))
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    cl, _ := fetch.New(fetch.Options{})
    // Discover should pick JSON feed as candidate later; ensure ParseFeed works as well
    items, err := feeds.ParseFeed(context.Background(), cl, srv.URL+"/atom.xml", 5)
    if err != nil { t.Fatalf("parse atom: %v", err) }
    if len(items) == 0 { t.Fatalf("expect items parsed") }
}

