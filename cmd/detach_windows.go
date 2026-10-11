//go:build windows

package cmd

import (
	"os/exec"
	"syscall"
)

// detach runs cmd in a process group of its own, so closing the console
// doesn't stop it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}
