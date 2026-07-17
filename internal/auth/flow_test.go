package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/config"
)

// fakeAuthServer emulates the 42 OAuth token endpoint.
func fakeAuthServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if got := r.Form.Get("code"); got != "test-code" {
			http.Error(w, fmt.Sprintf("unexpected code %q", got), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"fake-access","token_type":"bearer","refresh_token":"fake-refresh","expires_in":7200}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// browserSimulator returns an OpenBrowser hook that acts as the user:
// it extracts redirect_uri and state from the authorization URL and
// requests the local callback with the given query values.
func browserSimulator(t *testing.T, buildQuery func(state string) url.Values) func(string) error {
	t.Helper()
	return func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		redirect := parsed.Query().Get("redirect_uri")
		state := parsed.Query().Get("state")

		go func() {
			query := buildQuery(state)
			resp, err := http.Get(redirect + "?" + query.Encode())
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
}

func testConfig(serverURL string) config.Config {
	return config.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		APIBaseURL:   serverURL + "/v2",
		RedirectURI:  "http://127.0.0.1:53699/callback",
	}
}

func TestFlow_Login(t *testing.T) {
	server := fakeAuthServer(t)

	var sawAuthURL string
	flow := NewFlow(testConfig(server.URL), FlowOptions{
		OnAuthURL: func(u string) { sawAuthURL = u },
		OpenBrowser: browserSimulator(t, func(state string) url.Values {
			return url.Values{"code": {"test-code"}, "state": {state}}
		}),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := flow.Login(ctx)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token.AccessToken != "fake-access" {
		t.Errorf("AccessToken = %q, want fake-access", token.AccessToken)
	}
	if token.RefreshToken != "fake-refresh" {
		t.Errorf("RefreshToken = %q, want fake-refresh", token.RefreshToken)
	}
	if !strings.Contains(sawAuthURL, "/oauth/authorize") {
		t.Errorf("auth URL = %q, want authorize endpoint", sawAuthURL)
	}
	if !strings.Contains(sawAuthURL, "public") || !strings.Contains(sawAuthURL, "projects") {
		t.Errorf("auth URL = %q, want scopes public e projects", sawAuthURL)
	}
}

func TestFlow_Login_InvalidState(t *testing.T) {
	server := fakeAuthServer(t)

	flow := NewFlow(testConfig(server.URL), FlowOptions{
		OpenBrowser: browserSimulator(t, func(string) url.Values {
			return url.Values{"code": {"test-code"}, "state": {"forged-state"}}
		}),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := flow.Login(ctx); err == nil || !strings.Contains(err.Error(), "state inválido") {
		t.Fatalf("Login: err = %v, want state validation error", err)
	}
}

func TestFlow_Login_DeniedAuthorization(t *testing.T) {
	server := fakeAuthServer(t)

	flow := NewFlow(testConfig(server.URL), FlowOptions{
		OpenBrowser: browserSimulator(t, func(state string) url.Values {
			return url.Values{"error": {"access_denied"}, "state": {state}}
		}),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := flow.Login(ctx); err == nil || !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("Login: err = %v, want access_denied error", err)
	}
}

func TestFlow_Login_ContextTimeout(t *testing.T) {
	server := fakeAuthServer(t)

	flow := NewFlow(testConfig(server.URL), FlowOptions{
		OpenBrowser: func(string) error { return nil }, // user never authorizes
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if _, err := flow.Login(ctx); err == nil || !strings.Contains(err.Error(), "login não concluído") {
		t.Fatalf("Login: err = %v, want timeout error", err)
	}
}

func TestOAuthConfig_Endpoints(t *testing.T) {
	t.Parallel()

	cfg := OAuthConfig(config.Config{APIBaseURL: "https://api.intra.42.fr/v2"})
	if cfg.Endpoint.AuthURL != "https://api.intra.42.fr/oauth/authorize" {
		t.Errorf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://api.intra.42.fr/oauth/token" {
		t.Errorf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
}
