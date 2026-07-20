//go:build linux

package platform

import (
	"fmt"
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
	if _, valid := SampleProcessGroup(1 << 30); valid {
		t.Fatal("missing root process should produce an invalid sample")
	}
}

func TestScanProcDirentsVisitsOnlyNumericNames(t *testing.T) {
	var data []byte
	for _, name := range []string{".", "self", "123", "98765"} {
		length := (19 + len(name) + 1 + 7) &^ 7
		record := make([]byte, length)
		record[16], record[17] = byte(length), byte(length>>8)
		copy(record[19:], name)
		data = append(data, record...)
	}
	var pids []int
	if !scanProcDirents(data, func(pid int) { pids = append(pids, pid) }) {
		t.Fatal("valid dirents rejected")
	}
	if fmt.Sprint(pids) != "[123 98765]" {
		t.Fatalf("PIDs = %v", pids)
	}
	if scanProcDirents(data[:len(data)-1], func(int) {}) {
		t.Fatal("truncated dirent should be rejected")
	}
	missingNUL := make([]byte, 24)
	missingNUL[16] = byte(len(missingNUL))
	copy(missingNUL[19:], "12345")
	if scanProcDirents(missingNUL, func(int) {}) {
		t.Fatal("dirent without a NUL-terminated name should be rejected")
	}
	misaligned := make([]byte, 23)
	misaligned[16] = byte(len(misaligned))
	if scanProcDirents(misaligned, func(int) {}) {
		t.Fatal("unaligned dirent should be rejected")
	}
}

func TestSampleProcessGroupIncludesReparentedMembers(t *testing.T) {
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

	var child process
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
	metrics, valid := SampleProcessGroup(cmd.Process.Pid)
	if !valid || metrics.Processes < 2 {
		t.Fatalf("group metrics = %+v, valid = %v", metrics, valid)
	}
}

func TestParseProcessStatValidatesRequiredNumbers(t *testing.T) {
	fields := make([]string, 22)
	for index := range fields {
		fields[index] = "0"
	}
	fields[0], fields[1], fields[2] = "S", "42", "123"
	fields[17], fields[20], fields[21] = "7", "4096", "2"
	data := []byte(fmt.Sprintf("99 (name with ) parenthesis) %s", strings.Join(fields, " ")))
	process, err := parseProcessStat(data, 99)
	if err != nil {
		t.Fatal(err)
	}
	if process.PPID != 42 || process.GroupID != 123 || process.Threads != 7 ||
		process.VirtualBytes != 4096 || process.ResidentBytes == 0 {
		t.Fatalf("parsed process = %+v", process)
	}
	for _, field := range []int{1, 2, 17, 20, 21} {
		invalid := append([]string(nil), fields...)
		invalid[field] = "invalid"
		data = []byte(fmt.Sprintf("99 (name) %s", strings.Join(invalid, " ")))
		if _, err := parseProcessStat(data, 99); err == nil {
			t.Fatalf("field %d should reject malformed input", field)
		}
	}
}
