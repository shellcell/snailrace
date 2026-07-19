//go:build linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCountDirectoryDoesNotRequireSortedEntries(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"z", "a", "middle"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := countDirectory(directory); got != 3 {
		t.Fatalf("directory count = %d, want 3", got)
	}
}

func TestMissingRootIsNotAZeroSample(t *testing.T) {
	if _, valid := SampleTree(1 << 30); valid {
		t.Fatal("missing root process should produce an invalid sample")
	}
}

func TestSampleTreeIncludesReparentedProcessGroupMembers(t *testing.T) {
	childFile := filepath.Join(t.TempDir(), "child-pid")
	cmd := exec.Command(
		"sh", "-c",
		"(sleep 30 & printf '%s' $! > \"$1\"); exec sleep 30",
		"sh", childFile,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})

	var child Process
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(childFile)
		if err == nil {
			childPID, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr == nil {
				child, err = readProcess(childPID)
				if err == nil && child.PPID != cmd.Process.Pid {
					break
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if child.PID == 0 || child.PPID == cmd.Process.Pid || child.GroupID != cmd.Process.Pid {
		t.Fatalf("child process was not reparented in group: %+v", child)
	}
	metrics, valid := SampleTree(cmd.Process.Pid)
	if !valid || metrics.Processes < 2 {
		t.Fatalf("group metrics = %+v, valid = %v", metrics, valid)
	}
}
