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
	physicalValues := make([]float64, 0, len(runs))
	for _, run := range runs {
		if run.PhysicalFootprintValid {
			physicalValues = append(physicalValues, run.PeakPhysicalFootprintBytes)
		}
	}
	var physicalStats *Stats
	if len(physicalValues) > 0 {
		value := stats(physicalValues)
		physicalStats = &value
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
		PeakPhysicalFootprintBytes: physicalStats,
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
	finite := make([]float64, 0, len(values))
	for _, value := range values {
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			finite = append(finite, value)
		}
	}
	if len(finite) == 0 {
		return Stats{}
	}
	sorted := append([]float64(nil), finite...)
	sort.Float64s(sorted)
	mean := 0.0
	m2 := 0.0
	varianceOverflow := false
	for index, value := range sorted {
		count := float64(index + 1)
		previousMean := mean
		mean = previousMean*(1-1/count) + value/count
		if math.IsInf(mean, 0) {
			mean = math.Copysign(math.MaxFloat64, mean)
		}
		delta := value - previousMean
		term := delta * (value - mean)
		if math.IsInf(term, 0) || math.IsNaN(term) || math.MaxFloat64-m2 < term {
			varianceOverflow = true
			m2 = math.MaxFloat64
		} else if term > 0 {
			m2 += term
		}
	}
	standardDeviation := 0.0
	if len(sorted) > 1 && varianceOverflow {
		standardDeviation = math.MaxFloat64
	} else if len(sorted) > 1 {
		standardDeviation = math.Sqrt(m2 / float64(len(sorted)-1))
	}
	margin := 0.0
	validInterval := len(sorted) > 1
	if len(sorted) > 1 {
		margin = standardDeviation / math.Sqrt(float64(len(sorted))) *
			tCritical95(len(sorted)-1)
		if math.IsInf(margin, 0) {
			margin = math.MaxFloat64
		}
	}
	return Stats{
		N: len(sorted), Min: sorted[0], Max: sorted[len(sorted)-1], Mean: mean,
		StdDev: standardDeviation, Median: percentile(sorted, 0.5),
		P95: percentile(sorted, 0.95), CI95Low: finiteDifference(mean, margin),
		CI95High: finiteSum(mean, margin), CI95Valid: validInterval,
	}
}

func percentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := percentile * float64(len(sorted)-1)
	lower, upper := int(math.Floor(position)), int(math.Ceil(position))
	weight := position - float64(lower)
	return finiteSum(sorted[lower]*(1-weight), sorted[upper]*weight)
}

func finiteSum(left, right float64) float64 {
	result := left + right
	if math.IsInf(result, 0) {
		return math.Copysign(math.MaxFloat64, result)
	}
	return result
}

func finiteDifference(left, right float64) float64 {
	result := left - right
	if math.IsInf(result, 0) {
		return math.Copysign(math.MaxFloat64, result)
	}
	return result
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
