package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/nvizble/Lightyear42/internal/config"
	"golang.org/x/oauth2"
)

const successPage = `<!DOCTYPE html>
<html lang="pt-BR">
<head><meta charset="utf-8"><title>42 CLI</title></head>
<body style="font-family: sans-serif; text-align: center; margin-top: 4rem;">
  <h1>Login concluído</h1>
  <p>Você já pode fechar esta aba e voltar ao terminal.</p>
</body>
</html>`

// FlowOptions customizes UX hooks of the login flow.
// Hooks are injected by the command layer so the flow stays free of I/O to the terminal.
type FlowOptions struct {
	// OnAuthURL is called with the authorization URL before waiting for the callback.
	// Use it to show the URL to the user. Optional.
	OnAuthURL func(url string)
	// OpenBrowser opens the given URL in the user's browser.
	// Defaults to a platform-specific implementation. Optional.
	OpenBrowser func(url string) error
}

// Flow runs the OAuth2 Authorization Code flow with a temporary local
// HTTP server receiving the redirect callback.
type Flow struct {
	oauth *oauth2.Config
	opts  FlowOptions
}

// NewFlow creates a login flow from the CLI configuration.
func NewFlow(cfg config.Config, opts FlowOptions) *Flow {
	if opts.OpenBrowser == nil {
		opts.OpenBrowser = openBrowser
	}
	return &Flow{oauth: OAuthConfig(cfg), opts: opts}
}

type callbackResult struct {
	code string
	err  error
}

// Login executes the full flow: starts the callback server, directs the user
// to the authorization page and exchanges the received code for a token.
// The context bounds the whole operation; cancel it to abort the login.
func (f *Flow) Login(ctx context.Context) (*oauth2.Token, error) {
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("gerar state: %w", err)
	}

	redirect, err := url.Parse(f.oauth.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("redirect_uri inválida %q: %w", f.oauth.RedirectURL, err)
	}

	listener, err := net.Listen("tcp", redirect.Host)
	if err != nil {
		return nil, fmt.Errorf("abrir porta local %s (já em uso?): %w", redirect.Host, err)
	}

	results := make(chan callbackResult, 1)
	server := &http.Server{
		Handler:           callbackHandler(redirect.Path, state, results),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			results <- callbackResult{err: fmt.Errorf("servidor de callback: %w", err)}
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := f.oauth.AuthCodeURL(state)
	if f.opts.OnAuthURL != nil {
		f.opts.OnAuthURL(authURL)
	}
	// A browser failure is not fatal: the URL was already shown via OnAuthURL.
	_ = f.opts.OpenBrowser(authURL)

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("login não concluído: %w", ctx.Err())
	case res := <-results:
		if res.err != nil {
			return nil, res.err
		}
		token, err := f.oauth.Exchange(ctx, res.code)
		if err != nil {
			return nil, fmt.Errorf("trocar code por token: %w", err)
		}
		return token, nil
	}
}

// callbackHandler validates the OAuth redirect and delivers the code (or error).
func callbackHandler(path, state string, results chan<- callbackResult) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if errCode := query.Get("error"); errCode != "" {
			desc := query.Get("error_description")
			http.Error(w, "Autorização negada. Volte ao terminal.", http.StatusForbidden)
			results <- callbackResult{err: fmt.Errorf("autorização negada pela 42: %s %s", errCode, desc)}
			return
		}

		if query.Get("state") != state {
			http.Error(w, "State inválido.", http.StatusBadRequest)
			results <- callbackResult{err: errors.New("state inválido no callback: possível tentativa de CSRF, aborte e tente novamente")}
			return
		}

		code := query.Get("code")
		if code == "" {
			http.Error(w, "Code ausente.", http.StatusBadRequest)
			results <- callbackResult{err: errors.New("callback sem authorization code")}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(successPage))
		results <- callbackResult{code: code}
	})
	return mux
}

// randomState returns an unguessable string for CSRF protection.
func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
