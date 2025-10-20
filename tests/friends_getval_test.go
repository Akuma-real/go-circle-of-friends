package tests

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "go-circle-of-friends/internal/fetch"
    "go-circle-of-friends/internal/friends"
    "go-circle-of-friends/internal/rules"
)

func TestFriends_GetValFallbackAndAttr(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/l", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        _, _ = w.Write([]byte(`<!doctype html><ul>
        <li class="it" data-href="/x"><a class="nm2" href="/ok">NM</a><span class="nm1">X</span></li>
        </ul>`))
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()
    cl, _ := fetch.New(fetch.Options{})
    preset := rules.Preset{FriendsPage: &rules.FriendsPage{
        Item:   ".it",
        // name: missing .nm0 -> fallback .nm1 -> . (current text)
        Name:   ".nm0||.nm1||.",
        // link: prefer a@href else current element attribute data-href via @data-href
        Link:   "a@href||@data-href",
        Avatar: ".missing||img@src", // missing -> empty
    }}
    list, err := friends.ParseFriendsPage(context.Background(), cl, srv.URL+"/l", preset)
    if err != nil { t.Fatalf("parse: %v", err) }
    if len(list) != 1 { t.Fatalf("len=%d want=1", len(list)) }
    if list[0].Name != "X" { t.Fatalf("fallback name expected 'X', got %q", list[0].Name) }
    if list[0].Link == "" { t.Fatalf("link should not be empty") }
}

