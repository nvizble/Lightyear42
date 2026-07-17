package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// friendsKey is the config.yaml key holding the friends list.
const friendsKey = "friends"

// FriendsFile persists the user's friends list in the config file.
//
// It reads and writes only the config file (no env bindings, no defaults),
// so saving never leaks environment values like client_secret into the file.
type FriendsFile struct{}

// NewFriendsFile returns a friends store backed by config.yaml.
func NewFriendsFile() *FriendsFile {
	return &FriendsFile{}
}

// fileViper returns a Viper bound exclusively to the config file.
func (f *FriendsFile) fileViper() (*viper.Viper, Paths, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return nil, Paths{}, err
	}

	v := viper.New()
	v.SetConfigFile(paths.ConfigFile)
	v.SetConfigType("yaml")
	return v, paths, nil
}

// Load returns the stored friends list; empty when the file or key is absent.
func (f *FriendsFile) Load() ([]string, error) {
	v, _, err := f.fileViper()
	if err != nil {
		return nil, err
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, nil
		}
		// SetConfigFile bypasses ConfigFileNotFoundError; treat missing file as empty.
		if _, statErr := os.Stat(v.ConfigFileUsed()); os.IsNotExist(statErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("ler config: %w", err)
	}
	return v.GetStringSlice(friendsKey), nil
}

// Save writes the friends list, preserving the other keys of the file.
func (f *FriendsFile) Save(friends []string) error {
	v, paths, err := f.fileViper()
	if err != nil {
		return err
	}

	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("criar diretório de config: %w", err)
	}

	// Load existing keys so they are preserved on write; a missing file is fine.
	_ = v.ReadInConfig()

	v.Set(friendsKey, friends)
	if err := v.WriteConfigAs(paths.ConfigFile); err != nil {
		return fmt.Errorf("gravar config: %w", err)
	}
	return nil
}
