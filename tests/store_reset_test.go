package tests

import (
    "context"
    "path/filepath"
    "testing"

    "go-circle-of-friends/internal/model"
    store "go-circle-of-friends/internal/store"
)

func TestSQLite_Reset(t *testing.T) {
    dir := t.TempDir()
    dbpath := filepath.Join(dir, "t.db")
    s, err := store.OpenSQLite(dbpath)
    if err != nil { t.Fatalf("open: %v", err) }
    defer s.Close()
    ctx := context.Background()
    // seed
    if err := s.UpsertFriend(ctx, model.Friend{Name: "n", Link: "l"}); err != nil { t.Fatalf("seed friend: %v", err) }
    if err := s.UpsertPost(ctx, model.Post{Title: "t", Link: "p"}); err != nil { t.Fatalf("seed post: %v", err) }
    // reset
    if err := s.Reset(ctx); err != nil { t.Fatalf("reset: %v", err) }
    fr, _ := s.ListFriends(ctx)
    ps, _ := s.ListPosts(ctx)
    if len(fr) != 0 || len(ps) != 0 { t.Fatalf("not empty after reset: fr=%d ps=%d", len(fr), len(ps)) }
}

