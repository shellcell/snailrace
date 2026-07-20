package model

import "testing"

func BenchmarkSummarize1000Runs(b *testing.B) {
	runs := make([]Run, 1000)
	for index := range runs {
		value := float64((index * 7919) % len(runs))
		runs[index] = Run{
			WallSeconds: value, CPUUserSeconds: value, CPUSystemSeconds: value,
			AverageCPUPercent: value, PeakResidentBytes: value,
			OSMaxRSSBytes: value, MeanResidentBytes: value, PeakVirtualBytes: value,
			PeakProcesses: value, PeakThreads: value, PeakFileDescriptors: value,
			SampleCount: index, SampleCoverageSeconds: value,
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = Summarize(runs)
	}
}
