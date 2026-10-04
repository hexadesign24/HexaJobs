//go:build windows

package client

import (
	"os/exec"
	"syscall"
)

func configureBrowserCommand(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
