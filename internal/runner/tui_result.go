package runner

import (
	"os"
	"time"

	"github.com/shellcell/snailrace/internal/model"
)

func makeRun(
	state *os.ProcessState,
	elapsed time.Duration,
	user, system float64,
	rusageRSS uint64,
	samples sampleAggregate,
	reason string,
) model.Run {
	peak := samples.peak
	peak.Processes = max(peak.Processes, 1)
	peak.Threads = max(peak.Threads, 1)
	return model.Run{
		ExitCode: state.ExitCode(), WallSeconds: elapsed.Seconds(),
		CPUUserSeconds: user, CPUSystemSeconds: system,
		AverageCPUPercent:          averageCPUPercent(user, system, elapsed),
		PeakResidentBytes:          float64(peak.ResidentBytes),
		PeakPhysicalFootprintBytes: float64(peak.PhysicalFootprintBytes),
		PhysicalFootprintValid:     peak.PhysicalFootprintValid,
		OSMaxRSSBytes:              float64(rusageRSS),
		MeanResidentBytes:          samples.meanResident(),
		PeakVirtualBytes:           float64(peak.VirtualBytes),
		PeakProcesses:              float64(peak.Processes),
		PeakThreads:                float64(peak.Threads),
		PeakFileDescriptors:        float64(peak.FileDescriptors),
		StopReason:                 reason,
		SampleCount:                int(samples.sampleCount),
		SampleCoverageSeconds:      samples.sampleCoverageSeconds,
	}
}

func averageCPUPercent(user, system float64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return (user + system) / elapsed.Seconds() * 100
}
