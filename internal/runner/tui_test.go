package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
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
		context.Background(), Spec{Name: "true", Args: []string{"true"}},
		time.Millisecond, Options{TUI: true, Duration: 50 * time.Millisecond},
	)
	if err == nil {
		t.Fatal("early fixed-duration exit should invalidate the run")
	}
}

func TestFixedDurationTUIExcludesTerminationDelay(t *testing.T) {
	duration := 50 * time.Millisecond
	started := time.Now()
	run, err := runTUIOnce(
		context.Background(),
		Spec{Name: "ignore-term", Shell: "trap '' TERM; while :; do :; done"},
		5*time.Millisecond,
		Options{TUI: true, Duration: duration},
	)
	if err != nil {
		t.Fatal(err)
	}
	if runtime := time.Since(started); runtime > 2*time.Second {
		t.Fatalf("fixed-duration run took %s, teardown should be outside measurement", runtime)
	}
	if run.WallSeconds < duration.Seconds() || run.WallSeconds > 0.5 {
		t.Fatalf("wall time = %.3fs, want approximately %.3fs", run.WallSeconds, duration.Seconds())
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

func TestTerminalInputPumpStopsWithoutInput(t *testing.T) {
	input, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer inputWriter.Close()
	terminal, terminalReader, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	defer terminalReader.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go copyTerminalInput(ctx, terminal, int(input.Fd()), done)
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("terminal input pump survived cancellation")
	}
}

func TestInteractiveTUIUsesInjectedTerminalStreams(t *testing.T) {
	input, output, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer output.Close()
	_, err = runTUIOnce(
		context.Background(), Spec{Name: "sleep", Shell: "sleep 30"},
		5*time.Millisecond, Options{
			TUI: true, Interactive: true, Duration: 30 * time.Millisecond,
			Input: input, TerminalOut: output,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTerminalInputPumpCancelsWhileDestinationIsFull(t *testing.T) {
	input, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer inputWriter.Close()
	terminalReader, terminal, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer terminalReader.Close()
	defer terminal.Close()
	fillPipe(t, terminal)
	if _, err := inputWriter.Write([]byte("input")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go copyTerminalInput(ctx, terminal, int(input.Fd()), done)
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("terminal input pump blocked on a full destination")
	}
}

func TestTerminalOutputPumpCancelsWhileDestinationIsFull(t *testing.T) {
	source, sourceWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	defer sourceWriter.Close()
	outputReader, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer outputReader.Close()
	defer output.Close()
	fillPipe(t, output)
	if _, err := sourceWriter.Write([]byte("output")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go copyInteractiveTerminalOutput(
		ctx, int(source.Fd()), int(output.Fd()), done,
	)
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("terminal output pump blocked on a full destination")
	}
}

func fillPipe(t *testing.T, pipe *os.File) {
	t.Helper()
	buffer := []byte{0}
	for {
		poll := []unix.PollFd{{Fd: int32(pipe.Fd()), Events: unix.POLLOUT}}
		ready, err := unix.Poll(poll, 0)
		if err != nil {
			t.Fatal(err)
		}
		if ready == 0 {
			return
		}
		if _, err := unix.Write(int(pipe.Fd()), buffer); err != nil {
			t.Fatal(err)
		}
	}
}
