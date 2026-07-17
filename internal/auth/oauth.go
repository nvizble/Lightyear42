package auth

import (
	"strings"

	"github.com/joaodiniz/42cli/internal/config"
	"golang.org/x/oauth2"
)

// DefaultScopes are the OAuth scopes requested during login.
// "public" covers read access to intranet data; "projects" is required
// to open/close evaluation slots (POST/DELETE /v2/slots).
var DefaultScopes = []string{"public", "projects"}

// OAuthConfig builds the oauth2.Config for the 42 API from the CLI config.
//
// The OAuth endpoints live at the API root (https://api.intra.42.fr/oauth/...),
// while cfg.APIBaseURL points to the versioned REST base (.../v2), so the
// version suffix is stripped here. Keeping both derived from the same setting
// lets tests point everything at a fake server with a single override.
func OAuthConfig(cfg config.Config) *oauth2.Config {
	base := strings.TrimSuffix(strings.TrimSuffix(cfg.APIBaseURL, "/"), "/v2")

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Scopes:       DefaultScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   base + "/oauth/authorize",
			TokenURL:  base + "/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}
