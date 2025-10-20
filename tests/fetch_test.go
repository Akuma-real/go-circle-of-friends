package tests

import (
    "context"
    "errors"
    "net"
    "net/http"
    "net/http/httptest"
    "os"
    "sync/atomic"
    "testing"
    "time"

    fetch "go-circle-of-friends/internal/fetch"
)

func TestFetch_UserAgentAndSuccess(t *testing.T) {
    t.Setenv("COF_UA", "test-agent/1.0")
    var gotUA string
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotUA = r.Header.Get("User-Agent")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }))
    defer srv.Close()

    cl, err := fetch.New(fetch.Options{Timeout: 2 * time.Second})
    if err != nil { t.Fatalf("new client: %v", err) }
    resp, err := cl.Get(context.Background(), srv.URL)
    if err != nil { t.Fatalf("get: %v", err) }
    _ = resp.Body.Close()
    if gotUA != "test-agent/1.0" {
        t.Fatalf("user-agent = %q, want %q", gotUA, "test-agent/1.0")
    }
}

func TestFetch_RetryOnStatus(t *testing.T) {
    var calls int32
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        n := atomic.AddInt32(&calls, 1)
        if n == 1 {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }))
    defer srv.Close()

    cl, err := fetch.New(fetch.Options{Retry: 1, Timeout: 2 * time.Second})
    if err != nil { t.Fatalf("new client: %v", err) }
    resp, err := cl.Get(context.Background(), srv.URL)
    if err != nil { t.Fatalf("get: %v", err) }
    _ = resp.Body.Close()
    if n := atomic.LoadInt32(&calls); n != 2 {
        t.Fatalf("calls = %d, want 2", n)
    }
}

func TestFetch_Timeout(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(300 * time.Millisecond)
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()

    cl, err := fetch.New(fetch.Options{Timeout: 100 * time.Millisecond})
    if err != nil { t.Fatalf("new client: %v", err) }
    _, err = cl.Get(context.Background(), srv.URL)
    if err == nil { t.Fatal("expected timeout error, got nil") }
    if ne, ok := err.(net.Error); ok && ne.Timeout() { return }
    if !errors.Is(err, context.DeadlineExceeded) {
        t.Fatalf("unexpected error: %v", err)
    }
    _ = os.Unsetenv("COF_UA")
}

