package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

func TestKeyringStore_SaveLoadDelete(t *testing.T) {
	keyring.MockInit()
	store := NewKeyringStore()

	want := &oauth2.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "bearer",
		Expiry:       time.Now().Add(2 * time.Hour).Truncate(time.Second),
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.AccessToken != want.AccessToken {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, want.AccessToken)
	}
	if got.RefreshToken != want.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, want.RefreshToken)
	}
	if !got.Expiry.Equal(want.Expiry) {
		t.Errorf("Expiry = %v, want %v", got.Expiry, want.Expiry)
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNoToken) {
		t.Fatalf("Load after Delete: err = %v, want ErrNoToken", err)
	}
}

func TestKeyringStore_LoadWithoutToken(t *testing.T) {
	keyring.MockInit()
	store := NewKeyringStore()

	if _, err := store.Load(); !errors.Is(err, ErrNoToken) {
		t.Fatalf("Load: err = %v, want ErrNoToken", err)
	}
}

func TestKeyringStore_DeleteWithoutToken(t *testing.T) {
	keyring.MockInit()
	store := NewKeyringStore()

	if err := store.Delete(); !errors.Is(err, ErrNoToken) {
		t.Fatalf("Delete: err = %v, want ErrNoToken", err)
	}
}
