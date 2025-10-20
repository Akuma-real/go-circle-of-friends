package tests

import (
    "context"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "testing"
    "time"

    "go-circle-of-friends/internal/aggregate"
    "go-circle-of-friends/internal/config"
    "go-circle-of-friends/internal/fetch"
    "go-circle-of-friends/internal/rules"
    store "go-circle-of-friends/internal/store"
)

func TestAggregate_NormalMode_DBWriteAndClean(t *testing.T) {
    // mock server
    mux := http.NewServeMux()
    // static friends only, so no friend page
    mux.HandleFunc("/b1/index.xml", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/rss+xml")
        // very old pubDate to be cleaned
        _, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel>
        <title>b1</title><link>/b1</link>
        <item><title>old</title><link>http://ex/old</link><pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate></item>
        </channel></rss>`))
    })
    mux.HandleFunc("/b2/index.xml", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/rss+xml")
        // recent pubDate
        now := time.Now().UTC().Format(time.RFC1123)
        _, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel>
        <title>b2</title><link>/b2</link>
        <item><title>new</title><link>http://ex/new</link><pubDate>` + now + `</pubDate></item>
        </channel></rss>`))
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    // DB
    dir := t.TempDir()
    dbpath := filepath.Join(dir, "t.db")
    st, err := store.OpenSQLite(dbpath)
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    defer st.Close()

    cl, _ := fetch.New(fetch.Options{Timeout: 5 * time.Second})
    cfg := &config.Config{
        StaticFriends: []config.StaticFriend{
            {Name: "b1", Link: srv.URL + "/b1", FeedSuffix: "/index.xml"},
            {Name: "b2", Link: srv.URL + "/b2", FeedSuffix: "/index.xml"},
        },
        SimpleMode: false,
        OutdateCleanDays: 1,
        Concurrency: config.Concurrency{Fetch: 2, Retry: 0},
    }
    rl := &rules.Rules{}
    run := aggregate.New(cfg, st, cl, rl)
    if err := run.Run(context.Background()); err != nil { t.Fatalf("run: %v", err) }

    posts, err := st.ListPosts(context.Background())
    if err != nil { t.Fatalf("list posts: %v", err) }
    // old should be cleaned, only new remains
    if len(posts) != 1 || posts[0].Title != "new" { t.Fatalf("unexpected posts: %#v", posts) }
}

