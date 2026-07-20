package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/platform"
)

func runTUIOnce(
	ctx context.Context,
	spec commandSpec,
	interval time.Duration,
	options Options,
) (model.Run, error) {
	input := options.Input
	if input == nil {
		input = os.Stdin
	}
	output := options.TerminalOut
	if output == nil {
		output = os.Stdout
	}
	inputTerminal, inputIsTerminal := input.(interface{ Fd() uintptr })
	outputTerminal, outputIsTerminal := output.(interface{ Fd() uintptr })
	inputFile, inputIsFile := input.(*os.File)
	if options.Interactive && (!inputIsTerminal || !outputIsTerminal || !inputIsFile ||
		!term.IsTerminal(int(inputTerminal.Fd())) ||
		!term.IsTerminal(int(outputTerminal.Fd()))) {
		return model.Run{}, errors.New(
			"interactive TUI mode requires terminal stdin and stdout",
		)
	}
	size := terminalSize(inputFile, options.Width, options.Height)
	var state *term.State
	var err error
	if options.Interactive {
		state, err = term.MakeRaw(int(inputTerminal.Fd()))
		if err != nil {
			return model.Run{}, err
		}
		defer term.Restore(int(inputTerminal.Fd()), state)
	}

	cmd := spec.command(ctx)
	started := time.Now()
	terminal, err := pty.StartWithSize(cmd, size)
	if err != nil {
		return model.Run{}, err
	}
	defer terminal.Close()
	inputContext, stopInput := context.WithCancel(ctx)
	outputContext, stopOutput := context.WithCancel(ctx)
	defer stopInput()
	defer stopOutput()
	outputDone := make(chan struct{})
	var inputDone, resizeDone <-chan struct{}
	if options.Interactive {
		inputFinished := make(chan struct{})
		inputDone = inputFinished
		go copyTerminalInput(
			inputContext, terminal, int(inputTerminal.Fd()), inputFinished,
		)
		go copyInteractiveTerminalOutput(
			outputContext, int(terminal.Fd()), int(outputTerminal.Fd()), outputDone,
		)
		if options.FollowResize {
			resizeDone = followTerminalResize(inputContext, inputFile, terminal)
		}
	} else {
		go copyTerminalOutput(io.Discard, terminal, outputDone)
	}

	groupID := cmd.Process.Pid
	monitor := startMonitor(groupID, interval, platform.NewGroupSampler().Sample)
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	waitResult := waitForTUI(
		ctx, groupID, options.Duration, started, waitDone,
	)
	signalProcessGroup(groupID, syscall.SIGKILL)
	samples := monitor.finish()
	stopInput()
	if inputDone != nil {
		<-inputDone
	}
	if resizeDone != nil {
		<-resizeDone
	}
	select {
	case <-outputDone:
	case <-time.After(100 * time.Millisecond):
		stopOutput()
		_ = terminal.Close()
		<-outputDone
	}
	user, system, rusageRSS := platform.ResourceUsage(cmd.ProcessState)
	if ctx.Err() != nil {
		return model.Run{}, ctx.Err()
	}
	if options.Duration > 0 && waitResult.reason == "exited" {
		return model.Run{}, errors.New("TUI exited before the fixed duration")
	}
	if waitResult.err != nil && waitResult.reason == "exited" &&
		!isNonZeroExit(cmd, waitResult.err) {
		return model.Run{}, waitResult.err
	}
	return makeRun(
		cmd.ProcessState, waitResult.elapsed, user, system, rusageRSS,
		samples, waitResult.reason,
	), nil
}

type tuiWaitResult struct {
	err     error
	elapsed time.Duration
	reason  string
}

func waitForTUI(
	ctx context.Context,
	groupID int,
	duration time.Duration,
	started time.Time,
	waitDone <-chan error,
) tuiWaitResult {
	if duration == 0 {
		select {
		case err := <-waitDone:
			return tuiWaitResult{err: err, elapsed: time.Since(started), reason: "exited"}
		case <-ctx.Done():
			signalProcessGroup(groupID, syscall.SIGKILL)
			return tuiWaitResult{
				err: <-waitDone, elapsed: time.Since(started), reason: "interrupted",
			}
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
		return tuiWaitResult{err: err, elapsed: time.Since(started), reason: "exited"}
	case <-ctx.Done():
		signalProcessGroup(groupID, syscall.SIGKILL)
		return tuiWaitResult{
			err: <-waitDone, elapsed: time.Since(started), reason: "interrupted",
		}
	case <-timer.C:
		elapsed := time.Since(started)
		signalProcessGroup(groupID, syscall.SIGKILL)
		return tuiWaitResult{err: <-waitDone, elapsed: elapsed, reason: "duration"}
	}
}
