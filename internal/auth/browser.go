package auth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the URL in the default browser of the current platform.
func OpenBrowser(url string) error {
	return openWithSystem(url)
}

// OpenPath opens a local file with the platform default application (e.g. PDF viewer).
func OpenPath(path string) error {
	return openWithSystem(path)
}

func openWithSystem(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("abrir %s: %w", target, err)
	}
	return nil
}
