package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPreparationCancellationKillsDescendants(t *testing.T) {
	pidPath := t.TempDir() + "/child.pid"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunPreparation(
			ctx, fmt.Sprintf("sleep 30 & echo $! > %q; wait", pidPath), os.Stderr,
		)
	}()
	var pid int
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidPath)
		if err == nil {
			pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	if pid == 0 {
		cancel()
		<-done
		t.Fatal("preparation did not start its child")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("preparation error = %v, want cancellation", err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		if state, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil &&
			strings.Contains(string(state), ") Z ") {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("preparation child %d survived cancellation", pid)
}

func TestCancelledPreparationDoesNotStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	marker := t.TempDir() + "/started"
	err := RunPreparation(ctx, fmt.Sprintf("touch %q", marker), os.Stderr)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("preparation error = %v, want cancellation", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled preparation started: %v", err)
	}
}
