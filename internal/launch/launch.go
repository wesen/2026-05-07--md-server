// Package launch starts a terminal-independent copy of the desktop binary.
package launch

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// Result acknowledges process creation, not window readiness. PID may already
// have exited after a Wails single-instance handoff. LogPath holds child output.
type Result struct {
	PID     int
	LogPath string
}

// Start re-executes this binary in foreground mode, with detached process
// attributes and private log-backed output. It inherits the caller's cwd and
// environment, but never its terminal streams. It does not wait for the GUI.
func Start(file string, dark bool) (Result, error) {
	executable, err := os.Executable()
	if err != nil {
		return Result{}, errors.Wrap(err, "locate md-view executable")
	}
	args, err := childArgs(file, dark)
	if err != nil {
		return Result{}, err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return Result{}, errors.Wrap(err, "locate background log cache")
	}
	return start(exec.Command(executable, args...), filepath.Join(cache, "md-view"))
}

func childArgs(file string, dark bool) ([]string, error) {
	args := []string{"view", "--foreground"}
	if dark {
		args = append(args, "--dark")
	}
	if file != "" {
		abs, err := filepath.Abs(file)
		if err != nil {
			return nil, errors.Wrap(err, "resolve background file")
		}
		args = append(args, "--", abs)
	}
	return args, nil
}

// start is separate so process ownership can be tested without a WebView.
func start(cmd *exec.Cmd, logDir string) (Result, error) {
	if err := detach(cmd); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return Result{}, errors.Wrap(err, "create background log directory")
	}
	logFile, err := os.CreateTemp(logDir, "launch-*.log")
	if err != nil {
		return Result{}, errors.Wrap(err, "create background log")
	}
	defer func() { _ = logFile.Close() }()
	// nil stdin is /dev/null (or the Windows equivalent). Regular files avoid
	// os/exec copy goroutines and prevent inherited shell pipes staying open.
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close() // Windows cannot remove an open file.
		_ = os.Remove(logFile.Name())
		return Result{}, errors.Wrap(err, "start background desktop")
	}
	result := Result{PID: cmd.Process.Pid, LogPath: logFile.Name()}
	if err := cmd.Process.Release(); err != nil {
		return result, errors.Wrapf(err, "release background process %d (log %s)", result.PID, result.LogPath)
	}
	return result, nil
}
