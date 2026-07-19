package model

import (
	"encoding/json"
	"math"
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

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.12f, want %.12f", got, want)
	}
}
