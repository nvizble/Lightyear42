// Package auth implements OAuth2 authentication against the 42 Intra API
// and secure token persistence in the operating system keyring.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	keyringService = "42cli"
	keyringUser    = "oauth-token"
)

// ErrNoToken indicates that no token is stored (user is not logged in).
var ErrNoToken = errors.New("nenhum token salvo: execute `42 login`")

// TokenStore persists OAuth2 tokens securely.
type TokenStore interface {
	// Save stores the token, replacing any previous one.
	Save(token *oauth2.Token) error
	// Load returns the stored token, or ErrNoToken if none exists.
	Load() (*oauth2.Token, error)
	// Delete removes the stored token. Returns ErrNoToken if none exists.
	Delete() error
}

// KeyringStore stores tokens in the OS keyring (Keychain, Secret Service, WinCred).
type KeyringStore struct{}

// NewKeyringStore returns a TokenStore backed by the OS keyring.
func NewKeyringStore() *KeyringStore {
	return &KeyringStore{}
}

// Save serializes the token as JSON and writes it to the keyring.
func (s *KeyringStore) Save(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("serializar token: %w", err)
	}
	if err := keyring.Set(keyringService, keyringUser, string(data)); err != nil {
		return fmt.Errorf("salvar token no keyring: %w", err)
	}
	return nil
}

// Load reads and deserializes the token from the keyring.
func (s *KeyringStore) Load() (*oauth2.Token, error) {
	data, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrNoToken
		}
		return nil, fmt.Errorf("ler token do keyring: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("desserializar token: %w", err)
	}
	return &token, nil
}

// Delete removes the token from the keyring.
func (s *KeyringStore) Delete() error {
	if err := keyring.Delete(keyringService, keyringUser); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNoToken
		}
		return fmt.Errorf("remover token do keyring: %w", err)
	}
	return nil
}
