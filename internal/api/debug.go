package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/oauth2"
)

const maxDebugBodyBytes = 8192

// WithDebugLog enables verbose HTTP request/response logging to w.
// Authorization values are redacted. The logger wraps the transport so the
// final Authorization header (injected by oauth2) is visible (masked).
func WithDebugLog(w io.Writer) Option {
	return func(c *Client) {
		c.debug = w
		c.installDebugTransport()
	}
}

func (c *Client) installDebugTransport() {
	if c.debug == nil || c.http == nil {
		return
	}
	rt := c.http.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	if ot, ok := rt.(*oauth2.Transport); ok {
		base := ot.Base
		if base == nil {
			base = http.DefaultTransport
		}
		ot.Base = &debugRoundTripper{base: base, client: c}
		return
	}
	c.http.Transport = &debugRoundTripper{base: rt, client: c}
}

type debugRoundTripper struct {
	base   http.RoundTripper
	client *Client
}

func (d *debugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var reqBody []byte
	if req.GetBody != nil {
		if rc, err := req.GetBody(); err == nil {
			reqBody, _ = io.ReadAll(rc)
			_ = rc.Close()
		}
	}
	d.client.logRequest(req, reqBody)

	resp, err := d.base.RoundTrip(req)
	if err != nil {
		d.client.logResponse(nil, nil, err)
		return nil, err
	}

	respBody, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		d.client.logResponse(resp, nil, readErr)
		return nil, readErr
	}
	d.client.logResponse(resp, respBody, nil)
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	resp.ContentLength = int64(len(respBody))
	return resp, nil
}

func (c *Client) logf(format string, args ...any) {
	if c.debug == nil {
		return
	}
	fmt.Fprintf(c.debug, format, args...)
}

func (c *Client) logRequest(req *http.Request, body []byte) {
	if c.debug == nil {
		return
	}
	c.logf("\n=== HTTP REQUEST ===\n")
	c.logf("%s %s\n", req.Method, req.URL.String())
	c.logHeaders("Request-Headers", req.Header)
	if len(body) > 0 {
		c.logf("Request-Body (%d bytes):\n%s\n", len(body), truncateForDebug(body))
	} else {
		c.logf("Request-Body: <empty>\n")
	}
}

func (c *Client) logResponse(resp *http.Response, body []byte, err error) {
	if c.debug == nil {
		return
	}
	c.logf("=== HTTP RESPONSE ===\n")
	if err != nil {
		c.logf("Transport-Error: %v\n", err)
		c.logf("=====================\n\n")
		return
	}
	c.logf("Status: %s\n", resp.Status)
	c.logHeaders("Response-Headers", resp.Header)
	if len(body) > 0 {
		c.logf("Response-Body (%d bytes):\n%s\n", len(body), truncateForDebug(body))
	} else {
		c.logf("Response-Body: <empty>\n")
	}
	c.logf("=====================\n\n")
}

func (c *Client) logHeaders(title string, h http.Header) {
	c.logf("%s:\n", title)
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range h.Values(k) {
			c.logf("  %s: %s\n", k, redactHeaderValue(k, v))
		}
	}
}

func redactHeaderValue(key, value string) string {
	switch strings.ToLower(key) {
	case "authorization":
		return redactAuthorization(value)
	case "cookie", "set-cookie":
		if len(value) <= 12 {
			return "***"
		}
		return value[:4] + "***" + value[len(value)-4:]
	default:
		return value
	}
}

func redactAuthorization(value string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return "***"
	}
	token := strings.TrimSpace(value[len(prefix):])
	if len(token) <= 8 {
		return prefix + "***"
	}
	return prefix + token[:4] + "…" + token[len(token)-4:] + fmt.Sprintf(" (len=%d)", len(token))
}

func truncateForDebug(b []byte) string {
	if len(b) <= maxDebugBodyBytes {
		return string(b)
	}
	return string(b[:maxDebugBodyBytes]) + fmt.Sprintf("\n… truncated (%d more bytes)", len(b)-maxDebugBodyBytes)
}

// enrichErrorBody appends useful response headers (e.g. WWW-Authenticate on 403).
func enrichErrorBody(_ int, body string, h http.Header) string {
	var extra []string
	if v := h.Get("WWW-Authenticate"); v != "" {
		extra = append(extra, "WWW-Authenticate: "+v)
	}
	if v := h.Get("X-Error"); v != "" {
		extra = append(extra, "X-Error: "+v)
	}
	if v := h.Get("X-Application-Roles"); v != "" {
		extra = append(extra, "X-Application-Roles: "+v)
	}
	if len(extra) == 0 {
		return body
	}
	joined := strings.Join(extra, " | ")
	if body == "" {
		return joined
	}
	return body + " | " + joined
}
