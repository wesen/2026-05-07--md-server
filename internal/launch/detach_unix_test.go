//go:build linux || darwin

package launch

import (
	"testing"

	"golang.org/x/sys/unix"
)

func checkDetachedSession(t *testing.T, pid int) {
	t.Helper()
	sid, err := unix.Getsid(pid)
	if err != nil || sid != pid {
		t.Fatalf("child session = %d, want PID %d; error %v", sid, pid, err)
	}
}
