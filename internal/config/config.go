// Package config loads and exposes CLI configuration via Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	// AppName is the application name used for config directories and files.
	AppName = "42cli"

	// EnvPrefix is the environment variable prefix (e.g. FORTYTWO_CLIENT_ID).
	EnvPrefix = "FORTYTWO"

	// DefaultNtfyServer is the push server used when none is configured.
	DefaultNtfyServer = "https://ntfy.sh"
)

// Config holds runtime configuration for the CLI.
type Config struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	APIBaseURL   string `mapstructure:"api_base_url"`
	RedirectURI  string `mapstructure:"redirect_uri"`
	// Notifications configures push delivery for "lightyear notify".
	Notifications Notifications `mapstructure:"notifications"`
	// CampusLayout optionally describes the physical cluster grids of the
	// user's campus (key: cluster number as string). The API has no layout
	// endpoint, so without this the map is inferred from active sessions.
	CampusLayout map[string]ClusterLayout `mapstructure:"campus_layout"`
}

// Notifications holds the push notification settings. Today the only
// backend is ntfy (https://ntfy.sh): anyone who knows the topic name can
// read the notifications, so the topic doubles as the secret.
type Notifications struct {
	NtfyServer string `mapstructure:"ntfy_server"`
	NtfyTopic  string `mapstructure:"ntfy_topic"`
	// NtfyToken authenticates against protected topics; optional on ntfy.sh.
	NtfyToken string `mapstructure:"ntfy_token"`
}

// ClusterLayout is the grid size of one cluster. Seats is the real seat
// count for irregular clusters with gaps; when 0, rows × posts is assumed.
// NaturalPosts draws columns left-to-right (p1 … pN). The default map is
// mirrored (pN … p1) to match physical numbering on campuses like São Paulo.
// ReversePosts is kept for backward compatibility with v1.1.2 configs; when
// true it is ignored (mirror is already the default).
type ClusterLayout struct {
	Rows         int  `mapstructure:"rows"`
	Posts        int  `mapstructure:"posts"`
	Seats        int  `mapstructure:"seats"`
	NaturalPosts bool `mapstructure:"natural_posts"`
	ReversePosts bool `mapstructure:"reverse_posts"` // deprecated; default is mirrored
}

// Paths returns well-known filesystem locations for the CLI.
type Paths struct {
	ConfigDir  string
	ConfigFile string
	CacheDir   string
	DataDir    string
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		APIBaseURL:  "https://api.intra.42.fr/v2",
		RedirectURI: "http://127.0.0.1:53682/callback",
		Notifications: Notifications{
			NtfyServer: DefaultNtfyServer,
		},
	}
}

// ResolvePaths returns XDG-compliant paths for config, cache and data.
// On macOS, falls back to ~/Library when XDG_* is unset, matching common CLI practice
// while still honoring XDG_* when the user sets them.
func ResolvePaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	configDir = filepath.Join(configDir, AppName)

	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, AppName)

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}
	dataDir = filepath.Join(dataDir, AppName)

	return Paths{
		ConfigDir:  configDir,
		ConfigFile: filepath.Join(configDir, "config.yaml"),
		CacheDir:   cacheDir,
		DataDir:    dataDir,
	}, nil
}

// Load reads configuration from file, environment and defaults.
// Missing config files are not an error; defaults and env vars still apply.
func Load() (Config, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return Config{}, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(paths.ConfigDir)

	defaults := Default()
	v.SetDefault("client_id", defaults.ClientID)
	v.SetDefault("client_secret", defaults.ClientSecret)
	v.SetDefault("api_base_url", defaults.APIBaseURL)
	v.SetDefault("redirect_uri", defaults.RedirectURI)
	v.SetDefault("notifications.ntfy_server", defaults.Notifications.NtfyServer)

	v.SetEnvPrefix(EnvPrefix)
	v.AutomaticEnv()
	_ = v.BindEnv("client_id", EnvPrefix+"_CLIENT_ID")
	_ = v.BindEnv("client_secret", EnvPrefix+"_CLIENT_SECRET")
	_ = v.BindEnv("api_base_url", EnvPrefix+"_API_BASE_URL")
	_ = v.BindEnv("redirect_uri", EnvPrefix+"_REDIRECT_URI")
	_ = v.BindEnv("notifications.ntfy_server", EnvPrefix+"_NTFY_SERVER")
	_ = v.BindEnv("notifications.ntfy_topic", EnvPrefix+"_NTFY_TOPIC")
	_ = v.BindEnv("notifications.ntfy_token", EnvPrefix+"_NTFY_TOKEN")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

// EnsureConfigDir creates the config directory if it does not exist.
func EnsureConfigDir() error {
	paths, err := ResolvePaths()
	if err != nil {
		return err
	}
	return os.MkdirAll(paths.ConfigDir, 0o700)
}
