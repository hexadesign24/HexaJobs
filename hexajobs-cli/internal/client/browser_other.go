//go:build !windows

package client

import "os/exec"

func configureBrowserCommand(cmd *exec.Cmd) {}
