package auth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the URL in the default browser of the current platform.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("abrir navegador: %w", err)
	}
	return nil
}
