package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(serverURL string) *Client {
	return NewClient(serverURL, nil,
		WithMaxRetries(2),
		WithBaseBackoff(time.Millisecond),
	)
}

func TestClient_Get_DecodesJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me" {
			t.Errorf("path = %q, want /me", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"login":"jdiniz"}`))
	}))
	defer server.Close()

	var out struct {
		Login string `json:"login"`
	}
	if err := newTestClient(server.URL).Get(context.Background(), "/me", nil, &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out.Login != "jdiniz" {
		t.Errorf("Login = %q, want jdiniz", out.Login)
	}
}

func TestClient_Get_TypedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		wantErr error
	}{
		{"unauthorized", http.StatusUnauthorized, ErrUnauthorized},
		{"forbidden", http.StatusForbidden, ErrForbidden},
		{"not found", http.StatusNotFound, ErrNotFound},
		{"rate limited", http.StatusTooManyRequests, ErrRateLimited},
		{"server error", http.StatusBadGateway, ErrServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			err := newTestClient(server.URL).Get(context.Background(), "/x", nil, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Get_RetriesOn429ThenSucceeds(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var out struct {
		OK bool `json:"ok"`
	}
	if err := newTestClient(server.URL).Get(context.Background(), "/x", nil, &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !out.OK {
		t.Error("expected decoded body after retries")
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestClient_Get_DoesNotRetryOn404(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := newTestClient(server.URL).Get(context.Background(), "/x", nil, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 404)", got)
	}
}

func TestClient_Get_ExhaustsRetries(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := newTestClient(server.URL).Get(context.Background(), "/x", nil, nil)
	if !errors.Is(err, ErrServer) {
		t.Fatalf("err = %v, want ErrServer", err)
	}
}

func TestClient_Get_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, nil,
		WithMaxRetries(5),
		WithBaseBackoff(time.Hour), // força cancelamento durante o backoff
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.Get(ctx, "/x", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestClient_Post_SendsJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"user_id":42`) {
			t.Errorf("body = %s, want user_id 42", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer server.Close()

	var out []struct {
		ID int `json:"id"`
	}
	err := newTestClient(server.URL).Post(context.Background(), "/slots", map[string]any{
		"slot": map[string]any{"user_id": 42},
	}, &out)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if len(out) != 1 || out[0].ID != 1 {
		t.Errorf("out = %+v", out)
	}
}

func TestClient_Delete_NoContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/slots/99" {
			t.Errorf("%s %s, want DELETE /slots/99", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := newTestClient(server.URL).Delete(context.Background(), "/slots/99"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestClient_Get_QueryParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter[login]"); got != "jdiniz" {
			t.Errorf("filter[login] = %q, want jdiniz", got)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	query := map[string][]string{"filter[login]": {"jdiniz"}}
	if err := newTestClient(server.URL).Get(context.Background(), "/users", query, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
}
