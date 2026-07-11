package model

import (
	"math"
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
	if result.N != 5 {
		t.Fatalf("N = %d, want 5", result.N)
	}
}

func TestSingleObservationHasZeroWidthInterval(t *testing.T) {
	result := Summarize([]Run{{WallSeconds: 2.5}}).WallSeconds
	assertClose(t, result.StdDev, 0)
	assertClose(t, result.CI95Low, 2.5)
	assertClose(t, result.CI95High, 2.5)
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.12f, want %.12f", got, want)
	}
}
