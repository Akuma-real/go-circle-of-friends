package tests

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "go-circle-of-friends/internal/config"
    "go-circle-of-friends/internal/fetch"
    "go-circle-of-friends/internal/friends"
    "go-circle-of-friends/internal/rules"
)

func TestFriends_ParseFriendsPage(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.WriteHeader(200)
        _, _ = w.Write([]byte(`<!doctype html><ul>
            <li class="it"><a class="nm" href="/s1">A</a><img src="/a.png"></li>
            <li class="it"><a class="nm" href="/s2">B</a><img src="/b.png"></li>
        </ul>`))
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    cl, err := fetch.New(fetch.Options{})
    if err != nil { t.Fatalf("fetch client: %v", err) }

    preset := rules.Preset{FriendsPage: &rules.FriendsPage{
        Item: ".it",
        Name: ".nm",
        Link: "a@href",
        Avatar: "img@src",
    }}

    list, err := friends.ParseFriendsPage(context.Background(), cl, srv.URL+"/links", preset)
    if err != nil { t.Fatalf("parse friends: %v", err) }
    if len(list) != 2 { t.Fatalf("friends len=%d want=2", len(list)) }
    // 确认绝对化
    if list[0].Link == "/s1" || list[0].Avatar == "/a.png" {
        t.Fatalf("expect absolute urls, got link=%q avatar=%q", list[0].Link, list[0].Avatar)
    }
    // 基本字段
    if list[0].Name == "" || list[1].Name == "" {
        t.Fatalf("empty name in result")
    }
    _ = config.StaticFriend{}
}

