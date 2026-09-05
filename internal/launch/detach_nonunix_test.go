//go:build !linux && !darwin

package launch

import "testing"

// Native Windows console/session behavior requires a Windows smoke run.
func checkDetachedSession(_ *testing.T, _ int) {}
