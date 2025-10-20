package tests

import (
    "os"
    "path/filepath"
    "testing"

    "go-circle-of-friends/internal/config"
    "go-circle-of-friends/internal/rules"
)

func TestConfig_DefaultsAndValidate(t *testing.T) {
    dir := t.TempDir()
    f := filepath.Join(dir, "c.yaml")
    // Minimal valid config
    _ = os.WriteFile(f, []byte("LINK: []\nSETTINGS_FRIENDS_LINKS: []\nMAX_POSTS_NUM: 0\nOUTDATE_CLEAN: 0\nSIMPLE_MODE: true\n"), 0644)
    c, err := config.Load(f)
    if err != nil { t.Fatalf("load: %v", err) }
    if c.Database.Type != "sqlite" || c.Database.DSN == "" { t.Fatalf("defaults not applied: %+v", c.Database) }
    if c.LogFormat == "" || c.LogLocale == "" || c.LogColor == "" { t.Fatalf("log defaults missing") }

    // Negative numbers should error
    _ = os.WriteFile(f, []byte("MAX_POSTS_NUM: -1\nOUTDATE_CLEAN: 0\n"), 0644)
    if _, err := config.Load(f); err == nil { t.Fatalf("expect error for negative MAX_POSTS_NUM") }
}

func TestRules_GetPreset(t *testing.T) {
    r := &rules.Rules{Presets: map[string]rules.Preset{
        "Default": {FriendsPage: &rules.FriendsPage{Item: ".i"}},
        "clarity": {FriendsPage: &rules.FriendsPage{Item: ".c"}},
    }}
    p, ok := r.GetPreset("")
    if !ok || p.FriendsPage == nil || p.FriendsPage.Item == "" { t.Fatalf("default fallback failed") }
    p2, ok := r.GetPreset("DEFAULT")
    if !ok || p2.FriendsPage.Item != ".i" { t.Fatalf("case-insensitive lookup failed: %+v", p2) }
}

