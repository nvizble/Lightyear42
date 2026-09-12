package notify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewNtfy(t *testing.T) {
	tests := []struct {
		name       string
		server     string
		topic      string
		wantServer string
		wantErr    bool
	}{
		{name: "tópico vazio", topic: "", wantErr: true},
		{name: "só espaços", topic: "   ", wantErr: true},
		{name: "servidor padrão", topic: "abc", wantServer: DefaultNtfyServer},
		{name: "servidor custom", server: "https://push.exemplo.com/", topic: "abc", wantServer: "https://push.exemplo.com"},
		{name: "servidor inválido", server: "://quebrado", topic: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNtfy(tt.server, tt.topic, "")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewNtfy() = %+v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewNtfy: %v", err)
			}
			if got.Server() != tt.wantServer {
				t.Errorf("Server() = %q, want %q", got.Server(), tt.wantServer)
			}
		})
	}
}

func TestNewNtfyEmptyTopicIsErrNoTopic(t *testing.T) {
	if _, err := NewNtfy("", "", ""); !errors.Is(err, ErrNoTopic) {
		t.Fatalf("err = %v, want ErrNoTopic", err)
	}
}

func TestNtfySend(t *testing.T) {
	var got ntfyPayload
	var auth, contentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/" {
			t.Errorf("path = %s, want /", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		contentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender, err := NewNtfy(srv.URL, "topico-x", "tk_secreto", WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("NewNtfy: %v", err)
	}

	msg := Notification{
		Title:    "Você vai avaliar libft",
		Message:  "12/09 às 14:30 — joaodini",
		Click:    "https://profile.intra.42.fr/scale_teams/1/edit",
		Tags:     []string{"calendar"},
		Priority: PriorityHigh,
	}
	if err := sender.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if got.Topic != "topico-x" {
		t.Errorf("topic = %q, want %q", got.Topic, "topico-x")
	}
	// Accented, non-ASCII text must survive: that is why the JSON endpoint is
	// used instead of the header-based API.
	if got.Title != msg.Title {
		t.Errorf("title = %q, want %q", got.Title, msg.Title)
	}
	if got.Message != msg.Message {
		t.Errorf("message = %q, want %q", got.Message, msg.Message)
	}
	if got.Click != msg.Click {
		t.Errorf("click = %q, want %q", got.Click, msg.Click)
	}
	if got.Priority != PriorityHigh {
		t.Errorf("priority = %d, want %d", got.Priority, PriorityHigh)
	}
	if auth != "Bearer tk_secreto" {
		t.Errorf("Authorization = %q, want bearer token", auth)
	}
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
}

func TestNtfySendWithoutToken(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender, err := NewNtfy(srv.URL, "topico", "", WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("NewNtfy: %v", err)
	}
	if err := sender.Send(context.Background(), TestNotification()); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if auth != "" {
		t.Errorf("Authorization = %q, want empty", auth)
	}
}

func TestNtfySendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	sender, err := NewNtfy(srv.URL, "topico", "", WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("NewNtfy: %v", err)
	}

	err = sender.Send(context.Background(), TestNotification())
	if err == nil {
		t.Fatal("Send() = nil, want error")
	}
	if !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("err = %v, want status and body", err)
	}
}

func TestSubscribeURL(t *testing.T) {
	sender, err := NewNtfy("https://ntfy.sh/", "abc", "")
	if err != nil {
		t.Fatalf("NewNtfy: %v", err)
	}
	if got, want := sender.SubscribeURL(), "https://ntfy.sh/abc"; got != want {
		t.Errorf("SubscribeURL() = %q, want %q", got, want)
	}
}

func TestRandomTopic(t *testing.T) {
	first, err := RandomTopic()
	if err != nil {
		t.Fatalf("RandomTopic: %v", err)
	}
	second, err := RandomTopic()
	if err != nil {
		t.Fatalf("RandomTopic: %v", err)
	}

	if !strings.HasPrefix(first, TopicPrefix) {
		t.Errorf("topic = %q, want prefix %q", first, TopicPrefix)
	}
	if first == second {
		t.Error("RandomTopic() repeated the same topic")
	}
	if len(first) != len(TopicPrefix)+topicRandomBytes*2 {
		t.Errorf("len(topic) = %d, want %d", len(first), len(TopicPrefix)+topicRandomBytes*2)
	}
}
