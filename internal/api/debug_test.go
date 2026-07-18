package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPDebugLog_OnForbidden(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="42 API", error="insufficient scope"`)
		w.Header().Set("X-Application-Roles", "None")
		http.Error(w, `{"error":"Forbidden","message":"Insufficient scope"}`, http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	var buf bytes.Buffer
	client := NewClient(srv.URL, nil, WithHTTPClient(srv.Client()), WithDebugLog(&buf), WithMaxRetries(0))

	err := client.Get(context.Background(), "/projects/1/attachments", nil, &struct{}{})
	if err == nil {
		t.Fatal("esperava 403")
	}
	if !strings.Contains(err.Error(), "WWW-Authenticate") {
		t.Fatalf("erro sem WWW-Authenticate: %v", err)
	}

	log := buf.String()
	for _, want := range []string{
		"=== HTTP REQUEST ===",
		"GET ",
		"/projects/1/attachments",
		"=== HTTP RESPONSE ===",
		"403",
		"insufficient scope",
		"Insufficient scope",
	} {
		if !strings.Contains(strings.ToLower(log), strings.ToLower(want)) {
			t.Fatalf("log faltando %q:\n%s", want, log)
		}
	}
}

func TestRedactAuthorization(t *testing.T) {
	t.Parallel()

	got := redactAuthorization("Bearer abcdefghijklmnop")
	if strings.Contains(got, "abcdefghijklmnop") {
		t.Fatalf("token não mascarado: %s", got)
	}
	if !strings.HasPrefix(got, "Bearer ") {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateForDebug(t *testing.T) {
	t.Parallel()

	big := bytes.Repeat([]byte("a"), maxDebugBodyBytes+10)
	out := truncateForDebug(big)
	if !strings.Contains(out, "truncated") {
		t.Fatalf("esperado truncate: %s", out[:80])
	}
	_ = io.Discard
}
