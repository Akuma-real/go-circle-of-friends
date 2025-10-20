package tests

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "go-circle-of-friends/internal/aggregate"
    "go-circle-of-friends/internal/config"
    "go-circle-of-friends/internal/fetch"
    "go-circle-of-friends/internal/rules"
)

func TestAggregate_SimpleMode_BufferOnly(t *testing.T) {
    // mock server: friends page + two feeds
    mux := http.NewServeMux()
    mux.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        _, _ = w.Write([]byte(`<!doctype html><ul>
        <li class="f"><a href="`+"/s1"+`">S1</a><img src="/a.png"></li>
        <li class="f"><a href="`+"/s2"+`">S2</a><img src="/b.png"></li>
        </ul>`))
    })
    mux.HandleFunc("/s1/index.xml", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/rss+xml")
        _, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel>
        <title>s1</title><link>/s1</link>
        <item><title>p1</title><link>http://ex/1</link><pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate></item>
        </channel></rss>`))
    })
    mux.HandleFunc("/s2/atom.xml", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/atom+xml")
        _, _ = w.Write([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom">
        <title>s2</title><entry><title>p2</title><updated>2007-01-02T15:04:05Z</updated><link href="http://ex/2"/></entry>
        </feed>`))
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    cl, _ := fetch.New(fetch.Options{Timeout: 3 * time.Second})
    cfg := &config.Config{
        LinkSources: []config.LinkSource{{Type: "page", URL: srv.URL+"/links", Theme: "default"}},
        MaxPostsNum:  0,
        SimpleMode:   true,
        Concurrency:  config.Concurrency{Fetch: 4, Retry: 0},
    }
    rl := &rules.Rules{Presets: map[string]rules.Preset{
        "default": {FriendsPage: &rules.FriendsPage{Item: ".f", Name: ".", Link: "a@href", Avatar: "img@src"}},
    }}

    run := aggregate.New(cfg, nil, cl, rl)
    if err := run.Run(context.Background()); err != nil { t.Fatalf("run: %v", err) }
    fr, ps := run.BufferData()
    if len(fr) != 2 { t.Fatalf("friends=%d want=2", len(fr)) }
    if len(ps) < 2 { t.Fatalf("posts=%d want>=2", len(ps)) }
}

