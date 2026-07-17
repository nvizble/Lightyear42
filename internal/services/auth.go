// Package services implements business rules for the 42 CLI.
// Cobra commands call services; services never depend on Cobra.
package services

import (
	"context"
	"fmt"

	"github.com/nvizble/Lightyear42/internal/auth"
	"golang.org/x/oauth2"
)

// LoginFlow runs an interactive OAuth2 login and returns the obtained token.
type LoginFlow interface {
	Login(ctx context.Context) (*oauth2.Token, error)
}

// AuthService orchestrates authentication: login, logout and session state.
type AuthService struct {
	flow  LoginFlow
	store auth.TokenStore
}

// NewAuthService wires the login flow and token store dependencies.
func NewAuthService(flow LoginFlow, store auth.TokenStore) *AuthService {
	return &AuthService{flow: flow, store: store}
}

// Login authenticates the user and persists the token securely.
func (s *AuthService) Login(ctx context.Context) (*oauth2.Token, error) {
	token, err := s.flow.Login(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.store.Save(token); err != nil {
		return nil, fmt.Errorf("salvar credenciais: %w", err)
	}
	return token, nil
}

// Logout removes the stored token.
// Returns auth.ErrNoToken when there is no active session.
func (s *AuthService) Logout() error {
	return s.store.Delete()
}

// LoggedIn reports whether a token is stored locally.
// It does not validate the token against the API.
func (s *AuthService) LoggedIn() bool {
	_, err := s.store.Load()
	return err == nil
}
