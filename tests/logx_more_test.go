package tests

import (
    "bytes"
    "log/slog"
    "strings"
    "testing"

    "go-circle-of-friends/internal/logx"
)

func TestLogx_ErrorfAndColorAlways(t *testing.T) {
    // 确保未受 NO_COLOR 影响
    t.Setenv("NO_COLOR", "")
    out := captureStdout(func() {
        logx.Init("error", "pretty", "zh-CN", "always")
        logx.Errorf("boom %d", 1)
    })
    if !strings.Contains(out, "[错误]") {
        t.Fatalf("expect error label, got: %q", out)
    }
    if !strings.Contains(out, "\x1b[") {
        t.Fatalf("expect ansi color when color=always")
    }
}

func TestLogx_WithAttrsAndGroup(t *testing.T) {
    // use handler directly for attribute path
    var buf bytes.Buffer
    h := logx.NewPrettyHandler(&buf, slog.LevelInfo, "en", "never")
    logger := slog.New(h)
    logger = logger.With("k", "v").WithGroup("g")
    logger.Info("hello")
    s := buf.String()
    if !strings.Contains(s, "k=v") {
        t.Fatalf("expect flattened attr present, got: %q", s)
    }
}
