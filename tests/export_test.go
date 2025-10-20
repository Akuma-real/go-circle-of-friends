package tests

import (
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
    "time"

    "go-circle-of-friends/internal/export"
    "go-circle-of-friends/internal/model"
    store "go-circle-of-friends/internal/store"
)

func TestExport_ToJSON_WithCap(t *testing.T) {
    dir := t.TempDir()
    dbpath := filepath.Join(dir, "t.db")
    out := filepath.Join(dir, "out.json")
    s, err := store.OpenSQLite(dbpath)
    if err != nil { t.Fatalf("open: %v", err) }
    defer s.Close()
    ctx := context.Background()

    // Seed 200 posts with increasing Created
    now := time.Now()
    for i := 0; i < 200; i++ {
        p := model.Post{Title: "t", Link: "l"+string(rune(i+65)), Created: now.Add(time.Duration(i) * time.Minute)}
        if err := s.UpsertPost(ctx, p); err != nil { t.Fatalf("seed post: %v", err) }
    }
    if err := export.ToJSON(ctx, s, out); err != nil { t.Fatalf("export: %v", err) }
    b, _ := os.ReadFile(out)
    var e model.Export
    if err := json.Unmarshal(b, &e); err != nil { t.Fatalf("decode: %v", err) }
    if len(e.Posts) != 150 { t.Fatalf("len=%d want=150", len(e.Posts)) }
    // Ensure order new -> old
    if e.Posts[0].Created.Before(e.Posts[len(e.Posts)-1].Created) {
        t.Fatalf("order not desc")
    }
}

func TestExport_ToJSONData_WithStats(t *testing.T) {
    dir := t.TempDir()
    out := filepath.Join(dir, "out.json")
    friends := []model.Friend{{Name: "A", Link: "l", Avatar: "a"}, {Name: "B", Link: "l2", Avatar: "b", Error: "x"}}
    now := time.Now()
    posts := make([]model.Post, 0, 160)
    for i := 0; i < 160; i++ {
        posts = append(posts, model.Post{Title: "t", Link: "p"+string(rune(i+65)), Created: now.Add(time.Duration(i) * time.Minute)})
    }
    if err := export.ToJSONData(context.Background(), friends, posts, out); err != nil { t.Fatalf("export data: %v", err) }
    b, _ := os.ReadFile(out)
    var e model.Export
    if err := json.Unmarshal(b, &e); err != nil { t.Fatalf("decode: %v", err) }
    if e.Stats.FriendsTotal != 2 || e.Stats.FriendsAlive != 1 || e.Stats.FriendsError != 1 { t.Fatalf("stats friends mismatch: %+v", e.Stats) }
    if e.Stats.PostsTotal != 150 { t.Fatalf("stats posts_total=%d want=150", e.Stats.PostsTotal) }
}

