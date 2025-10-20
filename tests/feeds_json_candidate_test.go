package tests

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "go-circle-of-friends/internal/feeds"
    "go-circle-of-friends/internal/fetch"
)

func TestDiscoverFeed_JSONCandidate(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/feed+json")
        _, _ = w.Write([]byte(`{"version":"https://jsonfeed.org/version/1.1","items":[]}`))
    })
    // other paths 404
    srv := httptest.NewServer(mux)
    defer srv.Close()

    cl, _ := fetch.New(fetch.Options{})
    u, err := feeds.DiscoverFeed(context.Background(), cl, srv.URL+"/", "")
    if err != nil { t.Fatalf("discover: %v", err) }
    if want := srv.URL+"/index.json"; u != want { t.Fatalf("got=%q want=%q", u, want) }
}

