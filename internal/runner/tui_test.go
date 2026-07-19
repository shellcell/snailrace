package runner

import (
	"context"
	"testing"
	"time"
)

func TestFixedDurationTUIStopsProcessGroup(t *testing.T) {
	duration := 50 * time.Millisecond
	run, err := runTUIOnce(
		context.Background(),
		Spec{Name: "sleep", Shell: "sleep 5"},
		5*time.Millisecond,
		Options{TUI: true, Duration: duration},
	)
	if err != nil {
		t.Fatal(err)
	}
	if run.StopReason != "duration" {
		t.Fatalf("stop reason = %q, want duration", run.StopReason)
	}
	if run.WallSeconds < duration.Seconds() || run.WallSeconds > 0.2 {
		t.Fatalf("wall time = %.3fs, want approximately %.3fs", run.WallSeconds, duration.Seconds())
	}
}

func TestFixedDurationTUIRejectsEarlyExit(t *testing.T) {
	_, err := runTUIOnce(
		context.Background(), Spec{Name: "true", Args: []string{"/bin/true"}},
		time.Millisecond, Options{TUI: true, Duration: 50 * time.Millisecond},
	)
	if err == nil {
		t.Fatal("early fixed-duration exit should invalidate the run")
	}
}

func TestTUIRecordsNonZeroExit(t *testing.T) {
	run, err := runTUIOnce(
		context.Background(), Spec{Name: "exit", Shell: "exit 7"},
		time.Millisecond, Options{TUI: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if run.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", run.ExitCode)
	}
}

func TestTerminalSizeOverridesDimensions(t *testing.T) {
	width, height := ResolveTerminalSize(132, 43)
	if width != 132 || height != 43 {
		t.Fatalf("terminal size = %dx%d, want 132x43", width, height)
	}
}
