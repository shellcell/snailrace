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
	spec Spec,
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
	stop := make(chan struct{})
	done := make(chan platform.Metrics, 1)
	go monitor(cmd.Process.Pid, interval, stop, done)

	waitErr := cmd.Wait()
	elapsed := time.Since(started)
	signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
	close(stop)
	peak := <-done
	user, system, rusageRSS := platform.ResourceUsage(cmd.ProcessState)
	peak.Processes = max(peak.Processes, 1)
	peak.Threads = max(peak.Threads, 1)

	if ctx.Err() != nil {
		return model.Run{}, ctx.Err()
	}
	if waitErr != nil && !isNonZeroExit(cmd, waitErr) {
		return model.Run{}, waitErr
	}
	meanResident := sampledMeanResident(peak)
	return model.Run{
		ExitCode: cmd.ProcessState.ExitCode(), WallSeconds: elapsed.Seconds(),
		CPUUserSeconds: user, CPUSystemSeconds: system,
		AverageCPUPercent:          averageCPUPercent(user, system, elapsed),
		PeakResidentBytes:          float64(peak.ResidentBytes),
		PeakPhysicalFootprintBytes: float64(peak.PhysicalFootprintBytes),
		OSMaxRSSBytes:              float64(rusageRSS),
		MeanResidentBytes:          meanResident,
		PeakVirtualBytes:           float64(peak.VirtualBytes),
		PeakProcesses:              float64(peak.Processes),
		PeakThreads:                float64(peak.Threads),
		PeakFileDescriptors:        float64(peak.FileDescriptors),
		StopReason:                 "exited",
		SampleCount:                int(peak.SampleCount),
		SampleCoverageSeconds:      peak.SampleCoverageSeconds,
	}, nil
}

func isNonZeroExit(cmd *exec.Cmd, err error) bool {
	var exitError *exec.ExitError
	return errors.As(err, &exitError) && cmd.ProcessState != nil &&
		cmd.ProcessState.ExitCode() > 0
}

func monitor(
	pid int,
	interval time.Duration,
	stop <-chan struct{},
	done chan<- platform.Metrics,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var peak platform.Metrics
	var previousRSS uint64
	var previousAt time.Time
	addCoverage := func(now time.Time) {
		if previousAt.IsZero() {
			return
		}
		seconds := now.Sub(previousAt).Seconds()
		peak.ResidentByteSeconds += float64(previousRSS) * seconds
		peak.SampleCoverageSeconds += seconds
	}
	sample := func() {
		if current, valid := platform.SampleTree(pid); valid {
			now := time.Now()
			addCoverage(now)
			updatePeaks(&peak, current)
			previousRSS, previousAt = current.ResidentBytes, now
		}
	}
	if platform.SampleImmediately() {
		sample()
	}
	for {
		select {
		case <-ticker.C:
			sample()
		case <-stop:
			addCoverage(time.Now())
			done <- peak
			return
		}
	}
}

func updatePeaks(peak *platform.Metrics, current platform.Metrics) {
	peak.ResidentByteSamples += current.ResidentBytes
	peak.SampleCount++
	peak.ResidentBytes = max(peak.ResidentBytes, current.ResidentBytes)
	peak.PhysicalFootprintBytes = max(
		peak.PhysicalFootprintBytes, current.PhysicalFootprintBytes,
	)
	peak.VirtualBytes = max(peak.VirtualBytes, current.VirtualBytes)
	peak.Processes = max(peak.Processes, current.Processes)
	peak.Threads = max(peak.Threads, current.Threads)
	peak.FileDescriptors = max(
		peak.FileDescriptors, current.FileDescriptors,
	)
}

func sampledMeanResident(metrics platform.Metrics) float64 {
	if metrics.SampleCoverageSeconds > 0 {
		return metrics.ResidentByteSeconds / metrics.SampleCoverageSeconds
	}
	if metrics.SampleCount == 0 {
		return 0
	}
	return float64(metrics.ResidentByteSamples) / float64(metrics.SampleCount)
}

func averageCPUPercent(user, system float64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return (user + system) / elapsed.Seconds() * 100
}
