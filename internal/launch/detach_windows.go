package launch

import (
	"os/exec"
	"syscall"
)

func detach(cmd *exec.Cmd) error {
	const detachedProcess = 0x00000008
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return nil
}
