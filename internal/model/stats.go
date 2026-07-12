package model

import (
	"math"
	"sort"
)

func Summarize(runs []Run) Summary {
	values := func(pick func(Run) float64) []float64 {
		result := make([]float64, len(runs))
		for i, run := range runs {
			result[i] = pick(run)
		}
		return result
	}
	return Summary{
		WallSeconds: stats(values(func(r Run) float64 { return r.WallSeconds })),
		CPUTotalSeconds: stats(values(func(r Run) float64 {
			return r.CPUUserSeconds + r.CPUSystemSeconds
		})),
		CPUUserSeconds: stats(values(func(r Run) float64 {
			return r.CPUUserSeconds
		})),
		CPUSystemSeconds: stats(values(func(r Run) float64 {
			return r.CPUSystemSeconds
		})),
		AverageCPUPercent: stats(values(func(r Run) float64 {
			return r.AverageCPUPercent
		})),
		PeakResidentBytes: stats(values(func(r Run) float64 {
			return r.PeakResidentBytes
		})),
		OSMaxRSSBytes: stats(values(func(r Run) float64 {
			return r.OSMaxRSSBytes
		})),
		MeanResidentBytes: stats(values(func(r Run) float64 {
			return r.MeanResidentBytes
		})),
		PeakVirtualBytes: stats(values(func(r Run) float64 {
			return r.PeakVirtualBytes
		})),
		PeakProcesses: stats(values(func(r Run) float64 { return r.PeakProcesses })),
		PeakThreads:   stats(values(func(r Run) float64 { return r.PeakThreads })),
		PeakFileDescriptors: stats(values(func(r Run) float64 {
			return r.PeakFileDescriptors
		})),
		ValidSampleCount: stats(values(func(r Run) float64 {
			return float64(r.SampleCount)
		})),
		SampleCoverageSeconds: stats(values(func(r Run) float64 {
			return r.SampleCoverageSeconds
		})),
	}
}

func CalculateStats(values []float64) Stats { return stats(values) }

func stats(values []float64) Stats {
	if len(values) == 0 {
		return Stats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	total := 0.0
	for _, value := range sorted {
		total += value
	}
	mean := total / float64(len(sorted))
	variance := 0.0
	for _, value := range sorted {
		delta := value - mean
		variance += delta * delta
	}
	if len(sorted) > 1 {
		variance /= float64(len(sorted) - 1)
	}
	standardDeviation := math.Sqrt(variance)
	margin := 0.0
	validInterval := len(sorted) > 1
	if len(sorted) > 1 {
		margin = tCritical95(len(sorted)-1) * standardDeviation /
			math.Sqrt(float64(len(sorted)))
	}
	return Stats{
		N: len(sorted), Min: sorted[0], Max: sorted[len(sorted)-1], Mean: mean,
		StdDev: standardDeviation, Median: percentile(sorted, 0.5),
		P95: percentile(sorted, 0.95), CI95Low: mean - margin,
		CI95High: mean + margin, CI95Valid: validInterval,
	}
}

func percentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := percentile * float64(len(sorted)-1)
	lower, upper := int(math.Floor(position)), int(math.Ceil(position))
	weight := position - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// Two-sided 95% Student's t critical values for 1..30 degrees of freedom.
func tCritical95(degrees int) float64 {
	values := [...]float64{
		0, 12.706204736, 4.302652730, 3.182446305, 2.776445105,
		2.570581836, 2.446911851, 2.364624252, 2.306004135,
		2.262157163, 2.228138852, 2.200985160, 2.178812830,
		2.160368656, 2.144786688, 2.131449546, 2.119905299,
		2.109815578, 2.100922040, 2.093024054, 2.085963447,
		2.079613845, 2.073873068, 2.068657610, 2.063898562,
		2.059538553, 2.055529439, 2.051830516, 2.048407142,
		2.045229642, 2.042272456,
	}
	if degrees < len(values) {
		return values[degrees]
	}
	// Cornish-Fisher expansion around the normal 97.5th percentile.
	v := float64(degrees)
	z := 1.959963984540054
	z2 := z * z
	return z + (z*z2+z)/(4*v) +
		(5*z*z2*z2+16*z*z2+3*z)/(96*v*v) +
		(3*z*z2*z2*z2+19*z*z2*z2+17*z*z2-15*z)/(384*v*v*v)
}
