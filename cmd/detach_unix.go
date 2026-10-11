//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
)

// detach runs cmd in a session of its own, so it outlives the terminal it
// was started from and the terminal doesn't wait for it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
