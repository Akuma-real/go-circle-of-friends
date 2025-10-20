package tests

import (
    "context"
    "path/filepath"
    "testing"
    "time"

    "go-circle-of-friends/internal/model"
    store "go-circle-of-friends/internal/store"
)

func TestSQLite_MigrateCRUDAndClean(t *testing.T) {
    dir := t.TempDir()
    dbpath := filepath.Join(dir, "test.db")
    s, err := store.OpenSQLite(dbpath)
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    defer s.Close()

    ctx := context.Background()
    // Upsert friend (insert then update)
    f := model.Friend{Name: "Alice", Link: "https://a", Avatar: "av1"}
    if err := s.UpsertFriend(ctx, f); err != nil { t.Fatalf("upsert friend: %v", err) }
    f.Avatar = "av2"
    if err := s.UpsertFriend(ctx, f); err != nil { t.Fatalf("upsert friend upd: %v", err) }

    // Upsert posts (unique on link)
    old := time.Now().AddDate(-1, 0, 0)
    if err := s.UpsertPost(ctx, model.Post{Title: "o", Link: "l1", Created: old}); err != nil { t.Fatalf("upsert old: %v", err) }
    if err := s.UpsertPost(ctx, model.Post{Title: "n", Link: "l2", Created: time.Now()}); err != nil { t.Fatalf("upsert new: %v", err) }

    // List friends
    fr, err := s.ListFriends(ctx)
    if err != nil || len(fr) != 1 { t.Fatalf("list friends: %v len=%d", err, len(fr)) }
    if fr[0].Avatar != "av2" { t.Fatalf("friend not updated: %v", fr[0]) }

    // Stats
    st, err := s.Stats(ctx)
    if err != nil { t.Fatalf("stats: %v", err) }
    if st.FriendsTotal != 1 || st.PostsTotal != 2 { t.Fatalf("stats mismatch: %+v", st) }

    // Clean old posts (> 1 day)
    if err := s.CleanOldPosts(ctx, 1); err != nil { t.Fatalf("clean: %v", err) }
    posts, err := s.ListPosts(ctx)
    if err != nil { t.Fatalf("list posts: %v", err) }
    if len(posts) != 1 || posts[0].Title != "n" { t.Fatalf("unexpected posts after clean: %#v", posts) }
}

