//go:build unix

package ipc

import (
	"os/exec"
	"syscall"
)

// SetDetach starts the child in a new session (Setsid).
func SetDetach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
