//go:build !linux && !darwin && !windows

package launch

import (
	"os/exec"

	"github.com/pkg/errors"
)

func detach(_ *exec.Cmd) error {
	return errors.New("background launch is supported only on Linux, macOS, and Windows; use --foreground")
}
