package runner

import (
	"os/exec"
	"time"

	"perftool/internal/model"
	"perftool/internal/platform"
)

func makeTUIRun(
	cmd *exec.Cmd,
	elapsed time.Duration,
	user, system float64,
	rusageRSS uint64,
	peak platform.Metrics,
	reason string,
) model.Run {
	return model.Run{
		ExitCode: cmd.ProcessState.ExitCode(), WallSeconds: elapsed.Seconds(),
		CPUUserSeconds: user, CPUSystemSeconds: system,
		AverageCPUPercent:     averageCPUPercent(user, system, elapsed),
		PeakResidentBytes:     float64(peak.ResidentBytes),
		OSMaxRSSBytes:         float64(rusageRSS),
		MeanResidentBytes:     sampledMeanResident(peak),
		PeakVirtualBytes:      float64(peak.VirtualBytes),
		PeakProcesses:         float64(peak.Processes),
		PeakThreads:           float64(peak.Threads),
		PeakFileDescriptors:   float64(peak.FileDescriptors),
		StopReason:            reason,
		SampleCount:           int(peak.SampleCount),
		SampleCoverageSeconds: peak.SampleCoverageSeconds,
	}
}
