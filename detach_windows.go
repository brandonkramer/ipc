//go:build windows

package ipc

import (
	"os/exec"
	"syscall"
)

const flagDetachedProcess = 0x00000008 // DETACHED_PROCESS

// SetDetach starts the child detached from the launching console.
func SetDetach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | flagDetachedProcess,
	}
}
