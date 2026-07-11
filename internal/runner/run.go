package runner

import (
	"context"
	"os"
	"os/exec"
	"time"

	"perftool/internal/model"
	"perftool/internal/platform"
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
	close(stop)
	peak := <-done
	user, system, rusageRSS := platform.ResourceUsage(cmd.ProcessState)
	peak.ResidentBytes = max(peak.ResidentBytes, rusageRSS)
	peak.Processes = max(peak.Processes, 1)
	peak.Threads = max(peak.Threads, 1)

	if ctx.Err() != nil {
		return model.Run{}, ctx.Err()
	}
	if waitErr != nil {
		if _, ok := waitErr.(*exec.ExitError); !ok {
			return model.Run{}, waitErr
		}
	}
	meanResident := sampledMeanResident(peak)
	return model.Run{
		ExitCode: cmd.ProcessState.ExitCode(), WallSeconds: elapsed.Seconds(),
		CPUUserSeconds: user, CPUSystemSeconds: system,
		AverageCPUPercent:   averageCPUPercent(user, system, elapsed),
		PeakResidentBytes:   float64(peak.ResidentBytes),
		MeanResidentBytes:   meanResident,
		PeakVirtualBytes:    float64(peak.VirtualBytes),
		PeakProcesses:       float64(peak.Processes),
		PeakThreads:         float64(peak.Threads),
		PeakFileDescriptors: float64(peak.FileDescriptors),
		StopReason:          "exited",
	}, nil
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
	sample := func() {
		current := platform.SampleTree(pid)
		updatePeaks(&peak, current)
	}
	if platform.SampleImmediately() {
		sample()
	}
	for {
		select {
		case <-ticker.C:
			sample()
		case <-stop:
			done <- peak
			return
		}
	}
}

func updatePeaks(peak *platform.Metrics, current platform.Metrics) {
	peak.ResidentByteSamples += current.ResidentBytes
	peak.SampleCount++
	peak.ResidentBytes = max(peak.ResidentBytes, current.ResidentBytes)
	peak.VirtualBytes = max(peak.VirtualBytes, current.VirtualBytes)
	peak.Processes = max(peak.Processes, current.Processes)
	peak.Threads = max(peak.Threads, current.Threads)
	peak.FileDescriptors = max(
		peak.FileDescriptors, current.FileDescriptors,
	)
}

func sampledMeanResident(metrics platform.Metrics) float64 {
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
