package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// SaveCredentials writes OAuth client_id and client_secret to config.yaml,
// preserving friends, campus_layout and any other existing keys.
// When the file is new, api_base_url and redirect_uri defaults are also set.
func SaveCredentials(clientID, clientSecret string) (Paths, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if clientID == "" || clientSecret == "" {
		return Paths{}, fmt.Errorf("client_id e client_secret são obrigatórios")
	}

	paths, err := ResolvePaths()
	if err != nil {
		return Paths{}, err
	}
	if err := EnsureConfigDir(); err != nil {
		return Paths{}, fmt.Errorf("criar diretório de config: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(paths.ConfigFile)
	v.SetConfigType("yaml")
	fileExists := true
	if err := v.ReadInConfig(); err != nil {
		if _, statErr := os.Stat(paths.ConfigFile); os.IsNotExist(statErr) {
			fileExists = false
		} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Paths{}, fmt.Errorf("ler config: %w", err)
		} else {
			fileExists = false
		}
	}

	if !fileExists {
		defaults := Default()
		v.Set("api_base_url", defaults.APIBaseURL)
		v.Set("redirect_uri", defaults.RedirectURI)
	}

	v.Set("client_id", clientID)
	v.Set("client_secret", clientSecret)

	if err := v.WriteConfigAs(paths.ConfigFile); err != nil {
		return Paths{}, fmt.Errorf("gravar config: %w", err)
	}
	if err := os.Chmod(paths.ConfigFile, 0o600); err != nil {
		return Paths{}, fmt.Errorf("ajustar permissões do config: %w", err)
	}
	return paths, nil
}

// HasCredentials reports whether both OAuth credentials are set in cfg.
func HasCredentials(cfg Config) bool {
	return strings.TrimSpace(cfg.ClientID) != "" && strings.TrimSpace(cfg.ClientSecret) != ""
}
