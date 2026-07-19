package model

type SummaryAccumulator struct {
	wall, cpuTotal, cpuUser, cpuSystem RunningStats
	averageCPU                         RunningStats
	peakResident, physical, osMaxRSS   RunningStats
	meanResident, peakVirtual          RunningStats
	peakProcesses, peakThreads         RunningStats
	peakFileDescriptors                RunningStats
	validSampleCount, sampleCoverage   RunningStats
}

func NewSummaryAccumulator(capacity int) SummaryAccumulator {
	metric := func() RunningStats { return NewRunningStats(capacity) }
	return SummaryAccumulator{
		wall: metric(), cpuTotal: metric(), cpuUser: metric(), cpuSystem: metric(),
		averageCPU: metric(), peakResident: metric(), physical: metric(),
		osMaxRSS: metric(), meanResident: metric(), peakVirtual: metric(),
		peakProcesses: metric(), peakThreads: metric(), peakFileDescriptors: metric(),
		validSampleCount: metric(), sampleCoverage: metric(),
	}
}

func (summary *SummaryAccumulator) Add(run Run) {
	summary.wall.Add(run.WallSeconds)
	summary.cpuTotal.Add(run.CPUUserSeconds + run.CPUSystemSeconds)
	summary.cpuUser.Add(run.CPUUserSeconds)
	summary.cpuSystem.Add(run.CPUSystemSeconds)
	summary.averageCPU.Add(run.AverageCPUPercent)
	summary.peakResident.Add(run.PeakResidentBytes)
	if run.PhysicalFootprintValid {
		summary.physical.Add(run.PeakPhysicalFootprintBytes)
	}
	summary.osMaxRSS.Add(run.OSMaxRSSBytes)
	summary.meanResident.Add(run.MeanResidentBytes)
	summary.peakVirtual.Add(run.PeakVirtualBytes)
	summary.peakProcesses.Add(run.PeakProcesses)
	summary.peakThreads.Add(run.PeakThreads)
	summary.peakFileDescriptors.Add(run.PeakFileDescriptors)
	summary.validSampleCount.Add(float64(run.SampleCount))
	summary.sampleCoverage.Add(run.SampleCoverageSeconds)
}

func (summary SummaryAccumulator) Snapshot() Summary {
	result := Summary{
		WallSeconds:           summary.wall.Snapshot(),
		CPUTotalSeconds:       summary.cpuTotal.Snapshot(),
		CPUUserSeconds:        summary.cpuUser.Snapshot(),
		CPUSystemSeconds:      summary.cpuSystem.Snapshot(),
		AverageCPUPercent:     summary.averageCPU.Snapshot(),
		PeakResidentBytes:     summary.peakResident.Snapshot(),
		OSMaxRSSBytes:         summary.osMaxRSS.Snapshot(),
		MeanResidentBytes:     summary.meanResident.Snapshot(),
		PeakVirtualBytes:      summary.peakVirtual.Snapshot(),
		PeakProcesses:         summary.peakProcesses.Snapshot(),
		PeakThreads:           summary.peakThreads.Snapshot(),
		PeakFileDescriptors:   summary.peakFileDescriptors.Snapshot(),
		ValidSampleCount:      summary.validSampleCount.Snapshot(),
		SampleCoverageSeconds: summary.sampleCoverage.Snapshot(),
	}
	physical := summary.physical.Snapshot()
	if physical.N > 0 {
		result.PeakPhysicalFootprintBytes = &physical
	}
	return result
}
