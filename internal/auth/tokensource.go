package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/joaodiniz/42cli/internal/config"
	"golang.org/x/oauth2"
)

// NewTokenSource returns an oauth2.TokenSource that refreshes the stored
// token automatically and persists every refreshed token back to the store.
// Returns ErrNoToken when the user is not logged in.
//
// This is the integration point for the API client (Milestone 3): every
// authenticated request obtains a valid token from this source.
func NewTokenSource(ctx context.Context, cfg config.Config, store TokenStore) (oauth2.TokenSource, error) {
	token, err := store.Load()
	if err != nil {
		return nil, err
	}

	return &persistentTokenSource{
		src:   OAuthConfig(cfg).TokenSource(ctx, token),
		store: store,
		last:  token,
	}, nil
}

// persistentTokenSource wraps a refreshing TokenSource and saves new tokens.
type persistentTokenSource struct {
	src   oauth2.TokenSource
	store TokenStore

	mu   sync.Mutex
	last *oauth2.Token
}

// Token returns a valid token, persisting it if it was refreshed.
func (p *persistentTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	token, err := p.src.Token()
	if err != nil {
		return nil, fmt.Errorf("renovar token: %w", err)
	}

	if p.last == nil || token.AccessToken != p.last.AccessToken {
		if err := p.store.Save(token); err != nil {
			return nil, fmt.Errorf("persistir token renovado: %w", err)
		}
		p.last = token
	}
	return token, nil
}
