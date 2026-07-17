// Package api provides a typed HTTP client for the 42 Intra API (v2),
// with automatic token refresh, retries with exponential backoff and
// typed errors for the common failure modes.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	defaultMaxRetries  = 3
	defaultBaseBackoff = 500 * time.Millisecond
	defaultTimeout     = 30 * time.Second
	maxErrorBodyBytes  = 2048
)

// Client performs authenticated requests against the 42 API.
type Client struct {
	baseURL     string
	http        *http.Client
	maxRetries  int
	baseBackoff time.Duration
}

// Option customizes the Client.
type Option func(*Client)

// WithHTTPClient replaces the underlying HTTP client (useful in tests).
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.http = c }
}

// WithMaxRetries sets how many times a retryable request is re-attempted.
func WithMaxRetries(n int) Option {
	return func(cl *Client) { cl.maxRetries = n }
}

// WithBaseBackoff sets the initial backoff duration between retries.
func WithBaseBackoff(d time.Duration) Option {
	return func(cl *Client) { cl.baseBackoff = d }
}

// NewClient creates a client for the given base URL (e.g. https://api.intra.42.fr/v2).
// Requests are authenticated with tokens from source; pass nil for unauthenticated
// clients in tests.
func NewClient(baseURL string, source oauth2.TokenSource, opts ...Option) *Client {
	httpClient := &http.Client{Timeout: defaultTimeout}
	if source != nil {
		httpClient.Transport = &oauth2.Transport{Source: source}
	}

	client := &Client{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		http:        httpClient,
		maxRetries:  defaultMaxRetries,
		baseBackoff: defaultBaseBackoff,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// Get performs a GET request on path (e.g. "/me"), decoding the JSON
// response into out. Retryable failures (429, 5xx, network errors) are
// re-attempted with exponential backoff, honouring the Retry-After header.
func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := c.wait(ctx, attempt, lastErr); err != nil {
				return err
			}
		}

		body, err := c.do(ctx, endpoint)
		if err == nil {
			if out == nil {
				return nil
			}
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("decodificar resposta de %s: %w", path, err)
			}
			return nil
		}

		if !isRetryable(err) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("após %d tentativas: %w", c.maxRetries+1, lastErr)
}

// do executes a single GET request and returns the response body.
func (c *Client) do(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("montar requisição: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &transportError{err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ler resposta: %w", err)
		}
		return body, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	apiErr := newHTTPError(resp.StatusCode, strings.TrimSpace(string(body)))
	if resp.StatusCode == 429 {
		return nil, &rateLimitError{apiErr: apiErr, retryAfter: parseRetryAfter(resp.Header)}
	}
	return nil, apiErr
}

// wait sleeps before a retry, honouring Retry-After when present and the context.
func (c *Client) wait(ctx context.Context, attempt int, lastErr error) error {
	delay := c.backoff(attempt)

	var rle *rateLimitError
	if errors.As(lastErr, &rle) && rle.retryAfter > delay {
		delay = rle.retryAfter
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("requisição cancelada: %w", ctx.Err())
	case <-time.After(delay):
		return nil
	}
}

// backoff returns the exponential backoff with jitter for the given attempt.
func (c *Client) backoff(attempt int) time.Duration {
	base := c.baseBackoff * time.Duration(1<<(attempt-1))
	jitter := time.Duration(rand.Int64N(int64(base) / 2))
	return base + jitter
}

// transportError wraps network-level failures so they are retried.
type transportError struct{ err error }

func (t *transportError) Error() string { return "falha de rede: " + t.err.Error() }
func (t *transportError) Unwrap() error { return t.err }

// rateLimitError carries the Retry-After hint of a 429 response.
type rateLimitError struct {
	apiErr     *Error
	retryAfter time.Duration
}

func (r *rateLimitError) Error() string { return r.apiErr.Error() }
func (r *rateLimitError) Unwrap() error { return r.apiErr }

// isRetryable reports whether the request should be re-attempted.
func isRetryable(err error) bool {
	var te *transportError
	if errors.As(err, &te) {
		return true
	}
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrServer)
}

// parseRetryAfter reads the Retry-After header (seconds form).
func parseRetryAfter(h http.Header) time.Duration {
	seconds, err := strconv.Atoi(h.Get("Retry-After"))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
