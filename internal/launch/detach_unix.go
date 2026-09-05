//go:build linux || darwin

package launch

import (
	"os/exec"
	"syscall"
)

func detach(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return nil
}
