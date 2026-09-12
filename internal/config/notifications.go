package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// notificationsKey is the config.yaml section holding push settings.
const notificationsKey = "notifications"

// NotificationsFile persists the push notification settings in config.yaml.
//
// Like FriendsFile it reads and writes only the config file (no env bindings,
// no defaults), so saving never leaks environment values into the file.
type NotificationsFile struct{}

// NewNotificationsFile returns a notifications store backed by config.yaml.
func NewNotificationsFile() *NotificationsFile {
	return &NotificationsFile{}
}

// fileViper returns a Viper bound exclusively to the config file.
func fileViper() (*viper.Viper, Paths, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return nil, Paths{}, err
	}

	v := viper.New()
	v.SetConfigFile(paths.ConfigFile)
	v.SetConfigType("yaml")
	return v, paths, nil
}

// readFileConfig loads the config file into v, treating a missing file as empty.
func readFileConfig(v *viper.Viper) error {
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		// SetConfigFile bypasses ConfigFileNotFoundError; treat missing file as empty.
		if _, statErr := os.Stat(v.ConfigFileUsed()); os.IsNotExist(statErr) {
			return nil
		}
		return fmt.Errorf("ler config: %w", err)
	}
	return nil
}

// Load returns the stored notification settings; zero values when absent.
func (f *NotificationsFile) Load() (Notifications, error) {
	v, _, err := fileViper()
	if err != nil {
		return Notifications{}, err
	}
	if err := readFileConfig(v); err != nil {
		return Notifications{}, err
	}

	return Notifications{
		NtfyServer: v.GetString(notificationsKey + ".ntfy_server"),
		NtfyTopic:  v.GetString(notificationsKey + ".ntfy_topic"),
		NtfyToken:  v.GetString(notificationsKey + ".ntfy_token"),
	}, nil
}

// Save writes the notification settings, preserving the other keys of the file.
//
// The file is tightened to 0600 because the ntfy topic works as a shared
// secret: whoever knows it can read the notifications.
func (f *NotificationsFile) Save(n Notifications) error {
	v, paths, err := fileViper()
	if err != nil {
		return err
	}

	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("criar diretório de config: %w", err)
	}

	// Load existing keys so they are preserved on write; a missing file is fine.
	_ = v.ReadInConfig()

	v.Set(notificationsKey+".ntfy_server", n.NtfyServer)
	v.Set(notificationsKey+".ntfy_topic", n.NtfyTopic)
	v.Set(notificationsKey+".ntfy_token", n.NtfyToken)

	if err := v.WriteConfigAs(paths.ConfigFile); err != nil {
		return fmt.Errorf("gravar config: %w", err)
	}
	if err := os.Chmod(paths.ConfigFile, 0o600); err != nil {
		return fmt.Errorf("ajustar permissões do config: %w", err)
	}
	return nil
}
