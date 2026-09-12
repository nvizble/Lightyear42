package notify

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultNtfyServer is the public ntfy instance used when none is configured.
const DefaultNtfyServer = "https://ntfy.sh"

// ntfyTimeout bounds a single publish; the CLI must never hang on a push.
const ntfyTimeout = 10 * time.Second

// errBodyLimit caps how much of an error response we quote back to the user.
const errBodyLimit = 512

// ErrNoTopic means notifications are not configured yet.
var ErrNoTopic = errors.New("nenhum tópico ntfy configurado — rode: lightyear notify setup")

// Ntfy publishes notifications to an ntfy topic.
//
// It uses the JSON publish endpoint (POST / with a JSON body) instead of the
// header-based API, because titles and messages are written in Portuguese and
// HTTP headers are not a safe place for non-ASCII text.
type Ntfy struct {
	server string
	topic  string
	token  string
	client *http.Client
}

// NtfyOption customizes the sender.
type NtfyOption func(*Ntfy)

// WithHTTPClient replaces the default HTTP client (used in tests).
func WithHTTPClient(c *http.Client) NtfyOption {
	return func(n *Ntfy) { n.client = c }
}

// NewNtfy builds a sender for the given topic. An empty server falls back to
// DefaultNtfyServer; token is optional and only needed for protected topics.
func NewNtfy(server, topic, token string, opts ...NtfyOption) (*Ntfy, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, ErrNoTopic
	}

	server = strings.TrimRight(strings.TrimSpace(server), "/")
	if server == "" {
		server = DefaultNtfyServer
	}
	parsed, err := url.Parse(server)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("servidor ntfy inválido: %q", server)
	}

	n := &Ntfy{
		server: server,
		topic:  topic,
		token:  strings.TrimSpace(token),
		client: &http.Client{Timeout: ntfyTimeout},
	}
	for _, opt := range opts {
		opt(n)
	}
	return n, nil
}

// Topic returns the configured topic name.
func (n *Ntfy) Topic() string { return n.topic }

// Server returns the configured ntfy server root.
func (n *Ntfy) Server() string { return n.server }

// SubscribeURL is the address the user opens (or subscribes to in the app)
// to receive this sender's notifications.
func (n *Ntfy) SubscribeURL() string { return n.server + "/" + n.topic }

// ntfyPayload is the JSON publish body accepted by ntfy.
type ntfyPayload struct {
	Topic    string   `json:"topic"`
	Title    string   `json:"title,omitempty"`
	Message  string   `json:"message"`
	Click    string   `json:"click,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Priority int      `json:"priority,omitempty"`
}

// Send publishes one notification to the topic.
func (n *Ntfy) Send(ctx context.Context, msg Notification) error {
	body, err := json.Marshal(ntfyPayload{
		Topic:    n.topic,
		Title:    msg.Title,
		Message:  msg.Message,
		Click:    msg.Click,
		Tags:     msg.Tags,
		Priority: msg.Priority,
	})
	if err != nil {
		return fmt.Errorf("montar payload ntfy: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.server+"/", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("montar requisição ntfy: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if n.token != "" {
		req.Header.Set("Authorization", "Bearer "+n.token)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("enviar notificação para %s: %w", n.server, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, errBodyLimit))
		return fmt.Errorf("ntfy respondeu %s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, errBodyLimit))
	return nil
}

// topicRandomBytes sizes the random part of a generated topic. 12 bytes of
// entropy make the topic impractical to guess, which matters because on
// public ntfy servers the topic name is the only access control.
const topicRandomBytes = 12

// TopicPrefix marks generated topics as belonging to this CLI.
const TopicPrefix = "lightyear-"

// RandomTopic generates an unguessable topic name.
func RandomTopic() (string, error) {
	buf := make([]byte, topicRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("gerar tópico aleatório: %w", err)
	}
	return TopicPrefix + hex.EncodeToString(buf), nil
}
