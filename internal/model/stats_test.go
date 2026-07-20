package model

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestSummarizeUsesSampleStatistics(t *testing.T) {
	runs := []Run{
		{WallSeconds: 1}, {WallSeconds: 2}, {WallSeconds: 3},
		{WallSeconds: 4}, {WallSeconds: 5},
	}
	result := Summarize(runs).WallSeconds
	assertClose(t, result.Mean, 3)
	assertClose(t, result.StdDev, math.Sqrt(2.5))
	assertClose(t, result.Median, 3)
	assertClose(t, result.P95, 4.8)
	assertClose(t, result.CI95Low, 1.0367568385)
	assertClose(t, result.CI95High, 4.9632431615)
	if !result.CI95Valid {
		t.Fatal("five observations should define a confidence interval")
	}
	if result.N != 5 {
		t.Fatalf("N = %d, want 5", result.N)
	}
}

func TestSingleObservationHasNoConfidenceInterval(t *testing.T) {
	result := Summarize([]Run{{WallSeconds: 2.5}}).WallSeconds
	assertClose(t, result.StdDev, 0)
	if result.CI95Valid {
		t.Fatal("one observation cannot define a Student's t interval")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "ci95_low") {
		t.Fatalf("undefined confidence bounds should be omitted: %s", encoded)
	}
}

func TestUnavailablePhysicalFootprintIsOmittedFromJSON(t *testing.T) {
	summary := Summarize([]Run{{WallSeconds: 1}})
	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "peak_physical_footprint_bytes") {
		t.Fatalf("unavailable physical footprint should be omitted: %s", encoded)
	}
}

func TestPhysicalFootprintUsesExplicitPerRunValidity(t *testing.T) {
	summary := Summarize([]Run{
		{PeakPhysicalFootprintBytes: 0, PhysicalFootprintValid: true},
		{PeakPhysicalFootprintBytes: 10, PhysicalFootprintValid: true},
		{PeakPhysicalFootprintBytes: 1000, PhysicalFootprintValid: false},
	})
	if summary.PeakPhysicalFootprintBytes == nil {
		t.Fatal("valid zero-valued physical footprint should be retained")
	}
	if got := summary.PeakPhysicalFootprintBytes; got.N != 2 || got.Mean != 5 {
		t.Fatalf("physical footprint stats = %+v, want N=2 mean=5", *got)
	}
}

func TestStatsRemainFiniteForExtremeInputs(t *testing.T) {
	result := CalculateStats([]float64{
		math.MaxFloat64, -math.MaxFloat64, math.NaN(), math.Inf(1),
	})
	if result.N != 2 {
		t.Fatalf("N = %d, want two finite observations", result.N)
	}
	for name, value := range map[string]float64{
		"mean": result.Mean, "stddev": result.StdDev, "median": result.Median,
		"p95": result.P95, "ci low": result.CI95Low, "ci high": result.CI95High,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("%s is not finite: %v", name, value)
		}
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatal(err)
	}
}

func TestStatsJSONRoundTripPreservesConfidenceBounds(t *testing.T) {
	original := CalculateStats([]float64{1, 2, 4, 8})
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Stats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != original {
		t.Fatalf("round trip = %+v, want %+v", decoded, original)
	}
}

func TestStatsUnmarshalAcceptsLegacyConfidenceInterval(t *testing.T) {
	var decoded Stats
	if err := json.Unmarshal(
		[]byte(`{"n":2,"mean":3,"stddev":1,"ci95_valid":true}`), &decoded,
	); err != nil {
		t.Fatal(err)
	}
	if !decoded.CI95Valid || decoded.CI95Low == 0 || decoded.CI95High == 0 {
		t.Fatalf("legacy confidence interval = %+v", decoded)
	}
}

func TestStatsUnmarshalRejectsSingleObservationInterval(t *testing.T) {
	var decoded Stats
	if err := json.Unmarshal(
		[]byte(`{"n":1,"mean":3,"ci95_valid":true,"ci95_low":2,"ci95_high":4}`),
		&decoded,
	); err != nil {
		t.Fatal(err)
	}
	if decoded.CI95Valid {
		t.Fatal("one observation cannot have a confidence interval")
	}
}

func TestSummaryAccumulatorMatchesBatchSummaryAtEveryPrefix(t *testing.T) {
	runs := []Run{
		{WallSeconds: 1, CPUUserSeconds: 2, CPUSystemSeconds: 3,
			AverageCPUPercent: 4, PeakResidentBytes: 5, OSMaxRSSBytes: 6,
			MeanResidentBytes: 7, PeakVirtualBytes: 8, PeakProcesses: 9,
			PeakThreads: 10, PeakFileDescriptors: 11, SampleCount: 12,
			SampleCoverageSeconds: 13},
		{WallSeconds: 14, CPUUserSeconds: 15, CPUSystemSeconds: 16,
			AverageCPUPercent: 17, PeakResidentBytes: 18,
			PeakPhysicalFootprintBytes: 19, PhysicalFootprintValid: true,
			OSMaxRSSBytes: 20, MeanResidentBytes: 21, PeakVirtualBytes: 22,
			PeakProcesses: 23, PeakThreads: 24, PeakFileDescriptors: 25,
			SampleCount: 26, SampleCoverageSeconds: 27},
	}
	accumulator := NewSummaryAccumulator(len(runs))
	for index, run := range runs {
		accumulator.Add(run)
		got, want := accumulator.Snapshot(), independentSummary(runs[:index+1])
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("prefix %d summary mismatch:\ngot  %+v\nwant %+v", index+1, got, want)
		}
	}
}

func independentSummary(runs []Run) Summary {
	values := func(pick func(Run) float64) []float64 {
		result := make([]float64, len(runs))
		for index, run := range runs {
			result[index] = pick(run)
		}
		return result
	}
	physicalValues := make([]float64, 0, len(runs))
	for _, run := range runs {
		if run.PhysicalFootprintValid {
			physicalValues = append(physicalValues, run.PeakPhysicalFootprintBytes)
		}
	}
	var physical *Stats
	if len(physicalValues) > 0 {
		value := CalculateStats(physicalValues)
		physical = &value
	}
	return Summary{
		WallSeconds: CalculateStats(values(func(run Run) float64 { return run.WallSeconds })),
		CPUTotalSeconds: CalculateStats(values(func(run Run) float64 {
			return run.CPUUserSeconds + run.CPUSystemSeconds
		})),
		CPUUserSeconds:             CalculateStats(values(func(run Run) float64 { return run.CPUUserSeconds })),
		CPUSystemSeconds:           CalculateStats(values(func(run Run) float64 { return run.CPUSystemSeconds })),
		AverageCPUPercent:          CalculateStats(values(func(run Run) float64 { return run.AverageCPUPercent })),
		PeakResidentBytes:          CalculateStats(values(func(run Run) float64 { return run.PeakResidentBytes })),
		PeakPhysicalFootprintBytes: physical,
		OSMaxRSSBytes:              CalculateStats(values(func(run Run) float64 { return run.OSMaxRSSBytes })),
		MeanResidentBytes:          CalculateStats(values(func(run Run) float64 { return run.MeanResidentBytes })),
		PeakVirtualBytes:           CalculateStats(values(func(run Run) float64 { return run.PeakVirtualBytes })),
		PeakProcesses:              CalculateStats(values(func(run Run) float64 { return run.PeakProcesses })),
		PeakThreads:                CalculateStats(values(func(run Run) float64 { return run.PeakThreads })),
		PeakFileDescriptors:        CalculateStats(values(func(run Run) float64 { return run.PeakFileDescriptors })),
		ValidSampleCount:           CalculateStats(values(func(run Run) float64 { return float64(run.SampleCount) })),
		SampleCoverageSeconds: CalculateStats(values(func(run Run) float64 {
			return run.SampleCoverageSeconds
		})),
	}
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.12f, want %.12f", got, want)
	}
}
