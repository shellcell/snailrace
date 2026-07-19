package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/platform"
)

func runOnce(
	ctx context.Context,
	spec commandSpec,
	interval time.Duration,
	options Options,
) (model.Run, error) {
	if options.TUI {
		return runTUIOnce(ctx, spec, interval, options)
	}
	cmd := spec.command(ctx)
	configureProcessGroup(cmd)
	if options.ShowOutput {
		output := options.Output
		if output == nil {
			output = os.Stderr
		}
		cmd.Stdout, cmd.Stderr = output, output
	}
	started := time.Now()
	if err := cmd.Start(); err != nil {
		return model.Run{}, err
	}
	groupID := cmd.Process.Pid
	monitor := startMonitor(groupID, interval, platform.SampleProcessGroup)

	waitErr := cmd.Wait()
	elapsed := time.Since(started)
	signalProcessGroup(groupID, syscall.SIGKILL)
	samples := monitor.finish()
	user, system, rusageRSS := platform.ResourceUsage(cmd.ProcessState)

	if ctx.Err() != nil {
		return model.Run{}, ctx.Err()
	}
	if waitErr != nil && !isNonZeroExit(cmd, waitErr) {
		return model.Run{}, waitErr
	}
	return makeRun(
		cmd.ProcessState, elapsed, user, system, rusageRSS, samples, "exited",
	), nil
}

func isNonZeroExit(cmd *exec.Cmd, err error) bool {
	var exitError *exec.ExitError
	return errors.As(err, &exitError) && cmd.ProcessState != nil &&
		cmd.ProcessState.ExitCode() > 0
}
