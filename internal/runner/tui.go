package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"

	"perftool/internal/model"
	"perftool/internal/platform"
)

func runTUIOnce(
	ctx context.Context,
	spec Spec,
	interval time.Duration,
	options Options,
) (model.Run, error) {
	if options.Interactive && (!term.IsTerminal(int(os.Stdin.Fd())) ||
		!term.IsTerminal(int(os.Stdout.Fd()))) {
		return model.Run{}, errors.New(
			"interactive TUI mode requires terminal stdin and stdout",
		)
	}
	size := terminalSize(options.Width, options.Height)
	var state *term.State
	var err error
	if options.Interactive {
		state, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return model.Run{}, err
		}
		defer term.Restore(int(os.Stdin.Fd()), state)
	}

	cmd := spec.command(ctx)
	started := time.Now()
	terminal, err := pty.StartWithSize(cmd, size)
	if err != nil {
		return model.Run{}, err
	}
	defer terminal.Close()
	resizeContext, stopResize := context.WithCancel(ctx)
	defer stopResize()
	outputDone := make(chan struct{})
	if options.Interactive {
		go io.Copy(terminal, os.Stdin)
		go copyTerminalOutput(os.Stdout, terminal, outputDone)
		if options.FollowResize {
			followTerminalResize(resizeContext, terminal)
		}
	} else {
		go copyTerminalOutput(io.Discard, terminal, outputDone)
	}

	stopMonitor := make(chan struct{})
	monitorDone := make(chan platform.Metrics, 1)
	go monitor(cmd.Process.Pid, interval, stopMonitor, monitorDone)
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	waitErr, elapsed, reason := waitForTUI(
		ctx, cmd, options.Duration, started, waitDone,
	)
	signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
	close(stopMonitor)
	peak := <-monitorDone
	user, system, rusageRSS := platform.ResourceUsage(cmd.ProcessState)
	peak.Processes = max(peak.Processes, 1)
	peak.Threads = max(peak.Threads, 1)
	if ctx.Err() != nil {
		return model.Run{}, ctx.Err()
	}
	if options.Duration > 0 && reason == "exited" {
		return model.Run{}, errors.New("TUI exited before the fixed duration")
	}
	if waitErr != nil && reason == "exited" {
		return model.Run{}, waitErr
	}
	select {
	case <-outputDone:
	case <-time.After(100 * time.Millisecond):
	}
	return makeTUIRun(cmd, elapsed, user, system, rusageRSS, peak, reason), nil
}

func waitForTUI(
	ctx context.Context,
	cmd *exec.Cmd,
	duration time.Duration,
	started time.Time,
	waitDone <-chan error,
) (error, time.Duration, string) {
	if duration == 0 {
		select {
		case err := <-waitDone:
			return err, time.Since(started), "exited"
		case <-ctx.Done():
			signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			return <-waitDone, time.Since(started), "interrupted"
		}
	}
	remaining := time.Until(started.Add(duration))
	if remaining < 0 {
		remaining = 0
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case err := <-waitDone:
		return err, time.Since(started), "exited"
	case <-ctx.Done():
		signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
		return <-waitDone, time.Since(started), "interrupted"
	case <-timer.C:
		signalProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
		select {
		case err := <-waitDone:
			return err, time.Since(started), "duration"
		case <-time.After(time.Second):
			signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			return <-waitDone, time.Since(started), "duration"
		}
	}
}
