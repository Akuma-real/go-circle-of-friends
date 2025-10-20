package tests

import (
    "bytes"
    "io"
    "os"
    "strings"
    "testing"

    "go-circle-of-friends/internal/logx"
)

// captureStdout runs fn while capturing os.Stdout output and returns it as string.
func captureStdout(fn func()) string {
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w
    defer func() { os.Stdout = old }()
    fn()
    _ = w.Close()
    var buf bytes.Buffer
    _, _ = io.Copy(&buf, r)
    _ = r.Close()
    return buf.String()
}

func TestLogx_PrettyZH_Info(t *testing.T) {
    out := captureStdout(func() {
        logx.Init("debug", "pretty", "zh-CN", "never")
        logx.Infof("hello %s", "world")
    })
    if !strings.Contains(out, "[信息]") {
        t.Fatalf("expect zh label [信息], got: %q", out)
    }
}

func TestLogx_LevelFiltering(t *testing.T) {
    out := captureStdout(func() {
        logx.Init("warn", "pretty", "zh-CN", "never")
        logx.Infof("should not print")
        logx.Warnf("warn on")
    })
    if strings.Contains(out, "should not print") {
        t.Fatalf("info should be filtered when level=warn")
    }
    if !strings.Contains(out, "[警告]") {
        t.Fatalf("expect warn label present")
    }
}

func TestLogx_EnglishLabels(t *testing.T) {
    out := captureStdout(func() {
        logx.Init("info", "pretty", "en", "never")
        logx.Infof("ok")
    })
    if !strings.Contains(out, "[INFO]") {
        t.Fatalf("expect en label [INFO], got: %q", out)
    }
}

