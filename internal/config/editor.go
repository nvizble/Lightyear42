package config

import (
	"fmt"
	"os"
)

// colorschemeKey is where config.yaml keeps the editor's color scheme.
const colorschemeKey = "editor.colorscheme"

// Colorscheme is the embedded editor's saved color scheme ("" when none was
// chosen).
func Colorscheme() (string, error) {
	v, _, err := fileViper()
	if err != nil {
		return "", err
	}
	if err := readFileConfig(v); err != nil {
		return "", err
	}
	return v.GetString(colorschemeKey), nil
}

// SaveColorscheme stores the editor's color scheme, keeping the file's other
// keys (and its 0600: it may hold the OAuth client secret).
func SaveColorscheme(name string) error {
	v, paths, err := fileViper()
	if err != nil {
		return err
	}
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("criar diretório de config: %w", err)
	}
	if err := readFileConfig(v); err != nil {
		return err
	}
	v.Set(colorschemeKey, name)
	if err := v.WriteConfigAs(paths.ConfigFile); err != nil {
		return fmt.Errorf("gravar config: %w", err)
	}
	return os.Chmod(paths.ConfigFile, 0o600)
}
