package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/config"
	"golang.org/x/oauth2"
)

// memoryStore is an in-memory TokenStore for tests.
type memoryStore struct {
	token *oauth2.Token
	saves int
}

func (m *memoryStore) Save(token *oauth2.Token) error {
	m.token = token
	m.saves++
	return nil
}

func (m *memoryStore) Load() (*oauth2.Token, error) {
	if m.token == nil {
		return nil, ErrNoToken
	}
	return m.token, nil
}

func (m *memoryStore) Delete() error {
	if m.token == nil {
		return ErrNoToken
	}
	m.token = nil
	return nil
}

func TestNewTokenSource_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, err := NewTokenSource(context.Background(), config.Config{}, &memoryStore{})
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("err = %v, want ErrNoToken", err)
	}
}

func TestTokenSource_RefreshPersistsToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"new-access","token_type":"bearer","refresh_token":"new-refresh","expires_in":7200}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := &memoryStore{token: &oauth2.Token{
		AccessToken:  "expired-access",
		RefreshToken: "old-refresh",
		Expiry:       time.Now().Add(-time.Hour),
	}}

	cfg := config.Config{
		ClientID:     "id",
		ClientSecret: "secret",
		APIBaseURL:   server.URL + "/v2",
	}

	src, err := NewTokenSource(context.Background(), cfg, store)
	if err != nil {
		t.Fatalf("NewTokenSource: %v", err)
	}

	token, err := src.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", token.AccessToken)
	}
	if store.token.AccessToken != "new-access" {
		t.Errorf("stored AccessToken = %q, want new-access (refresh must persist)", store.token.AccessToken)
	}

	// A second call with a still-valid token must not persist again.
	saves := store.saves
	if _, err := src.Token(); err != nil {
		t.Fatalf("Token (2nd): %v", err)
	}
	if store.saves != saves {
		t.Errorf("saves = %d, want %d (valid token must not re-save)", store.saves, saves)
	}
}
