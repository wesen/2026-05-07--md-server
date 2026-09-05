package launch

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestChildArgs(t *testing.T) {
	for _, file := range []string{"", "notes.md", "a b;$(echo nope).md", "--dark", "view", filepath.Join(t.TempDir(), "absolute.md")} {
		for _, dark := range []bool{false, true} {
			got, err := childArgs(file, dark)
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"view", "--foreground"}
			if dark {
				want = append(want, "--dark")
			}
			if file != "" {
				abs, err := filepath.Abs(file)
				if err != nil {
					t.Fatal(err)
				}
				want = append(want, "--", abs)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %q, want %q", got, want)
			}
		}
	}
}

type observation struct {
	PID   int
	CWD   string
	Token string
	Args  []string
	EOF   bool
}

// TestLaunchHelper is re-executed as a real detached process. The deadline
// prevents leaked helpers even if the parent test is interrupted.
func TestLaunchHelper(t *testing.T) {
	if os.Getenv("MDV_LAUNCH_HELPER") != "1" {
		return
	}
	cwd, _ := os.Getwd()
	_, err := io.ReadAll(os.Stdin)
	data, _ := json.Marshal(observation{
		PID: os.Getpid(), CWD: cwd, Token: os.Getenv("MDV_LAUNCH_TOKEN"),
		Args: os.Args, EOF: err == nil,
	})
	fmt.Println(string(data))
	fmt.Fprintln(os.Stderr, "helper stderr")
	gate := os.Getenv("MDV_LAUNCH_GATE")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(gate); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func requireSupported(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("unsupported desktop platform")
	}
}

func TestDetachedProcess(t *testing.T) {
	requireSupported(t)
	dir := t.TempDir()
	gate := filepath.Join(dir, "exit")
	t.Setenv("MDV_LAUNCH_HELPER", "1")
	t.Setenv("MDV_LAUNCH_TOKEN", "inherited value")
	t.Setenv("MDV_LAUNCH_GATE", gate)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestLaunchHelper$", "--", "a b;$(no-shell).md")
	result, err := start(cmd, filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(gate, nil, 0600)
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			data, _ := os.ReadFile(result.LogPath)
			if strings.Contains(string(data), "PASS") {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Error("helper did not exit after gate opened")
	})
	checkDetachedSession(t, result.PID)
	var data []byte
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, _ = os.ReadFile(result.LogPath)
		if strings.Contains(string(data), "helper stderr") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	var got observation
	if err := json.Unmarshal([]byte(strings.SplitN(string(data), "\n", 2)[0]), &got); err != nil {
		t.Fatalf("helper output %q: %v", data, err)
	}
	cwd, _ := os.Getwd()
	if got.PID != result.PID || got.CWD != cwd || got.Token != "inherited value" || !got.EOF {
		t.Fatalf("incorrect inherited state: %+v; result=%+v cwd=%q", got, result, cwd)
	}
	if !reflect.DeepEqual(got.Args, cmd.Args) {
		t.Fatalf("args = %q, want %q", got.Args, cmd.Args)
	}
	if !strings.Contains(string(data), "helper stderr") || strings.Contains(string(data), "PASS") {
		t.Fatalf("child should be alive with redirected stderr: %q", data)
	}
	info, err := os.Stat(result.LogPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("log permissions = %v", info.Mode())
	}
}

func TestStartFailureCleansLog(t *testing.T) {
	requireSupported(t)
	dir := t.TempDir()
	_, err := start(exec.Command(filepath.Join(dir, "missing-executable")), dir)
	if err == nil || !strings.Contains(err.Error(), "start background desktop") {
		t.Fatalf("error = %v", err)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 0 {
		t.Fatalf("unused logs: %v, error %v", files, err)
	}
}

func TestLogDirectoryFailure(t *testing.T) {
	requireSupported(t)
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := start(exec.Command("unused"), filepath.Join(path, "logs"))
	if err == nil || !strings.Contains(err.Error(), "create background log directory") {
		t.Fatalf("error = %v", err)
	}
}
