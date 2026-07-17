package services

import (
	"context"
	"errors"
	"testing"

	"github.com/nvizble/Lightyear42/internal/auth"
	"golang.org/x/oauth2"
)

type fakeFlow struct {
	token *oauth2.Token
	err   error
}

func (f *fakeFlow) Login(context.Context) (*oauth2.Token, error) {
	return f.token, f.err
}

type fakeStore struct {
	token   *oauth2.Token
	saveErr error
}

func (s *fakeStore) Save(token *oauth2.Token) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.token = token
	return nil
}

func (s *fakeStore) Load() (*oauth2.Token, error) {
	if s.token == nil {
		return nil, auth.ErrNoToken
	}
	return s.token, nil
}

func (s *fakeStore) Delete() error {
	if s.token == nil {
		return auth.ErrNoToken
	}
	s.token = nil
	return nil
}

func TestAuthService_Login(t *testing.T) {
	t.Parallel()

	errFlow := errors.New("flow failed")
	errSave := errors.New("keyring unavailable")

	tests := []struct {
		name      string
		flow      *fakeFlow
		store     *fakeStore
		wantErr   error
		wantSaved bool
	}{
		{
			name:      "success saves token",
			flow:      &fakeFlow{token: &oauth2.Token{AccessToken: "abc"}},
			store:     &fakeStore{},
			wantSaved: true,
		},
		{
			name:    "flow error is returned",
			flow:    &fakeFlow{err: errFlow},
			store:   &fakeStore{},
			wantErr: errFlow,
		},
		{
			name:    "store error is returned",
			flow:    &fakeFlow{token: &oauth2.Token{AccessToken: "abc"}},
			store:   &fakeStore{saveErr: errSave},
			wantErr: errSave,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewAuthService(tt.flow, tt.store)
			_, err := svc.Login(context.Background())

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Login: err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login: %v", err)
			}
			if tt.wantSaved && tt.store.token == nil {
				t.Fatal("token was not saved")
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	t.Parallel()

	t.Run("removes stored token", func(t *testing.T) {
		t.Parallel()
		store := &fakeStore{token: &oauth2.Token{AccessToken: "abc"}}
		svc := NewAuthService(nil, store)

		if err := svc.Logout(); err != nil {
			t.Fatalf("Logout: %v", err)
		}
		if store.token != nil {
			t.Fatal("token still stored after logout")
		}
	})

	t.Run("no session returns ErrNoToken", func(t *testing.T) {
		t.Parallel()
		svc := NewAuthService(nil, &fakeStore{})

		if err := svc.Logout(); !errors.Is(err, auth.ErrNoToken) {
			t.Fatalf("Logout: err = %v, want ErrNoToken", err)
		}
	})
}

func TestAuthService_LoggedIn(t *testing.T) {
	t.Parallel()

	svc := NewAuthService(nil, &fakeStore{token: &oauth2.Token{AccessToken: "abc"}})
	if !svc.LoggedIn() {
		t.Fatal("LoggedIn = false, want true")
	}

	svc = NewAuthService(nil, &fakeStore{})
	if svc.LoggedIn() {
		t.Fatal("LoggedIn = true, want false")
	}
}
