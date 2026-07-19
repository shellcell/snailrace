package runner

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/shellcell/snailrace/internal/model"
)

func TestBenchmarkProgressIncludesRunningEstimate(t *testing.T) {
	var events []ProgressEvent
	_, err := Benchmark(
		context.Background(),
		[]Spec{{Name: "sleep", Args: []string{"/bin/sleep", "0.002"}}},
		Config{Runs: 2, Interval: time.Millisecond},
		Options{Progress: func(event ProgressEvent) { events = append(events, event) }},
	)
	if err != nil {
		t.Fatal(err)
	}
	foundEstimate, foundRace, foundFinish := false, false, false
	for _, event := range events {
		foundEstimate = foundEstimate || event.HasEstimate
		foundRace = foundRace ||
			len(event.Estimates) == 1 && event.Estimates[0].HasEstimate
		foundFinish = foundFinish || event.Finished
	}
	if !foundEstimate || !foundRace || !foundFinish {
		t.Fatalf(
			"progress estimate/race/finish = %v/%v/%v",
			foundEstimate, foundRace, foundFinish,
		)
	}
}

func BenchmarkProgressUpdates1000Runs(b *testing.B) {
	runs := make([]model.Run, 1000)
	for index := range runs {
		runs[index] = model.Run{
			Index: index + 1, WallSeconds: float64(index+1) / 1000,
			CPUUserSeconds: 0.01, PeakResidentBytes: float64(index+1) * 1024,
			MeanResidentBytes: float64(index+1) * 512,
			SampleCount:       10, SampleCoverageSeconds: 0.1,
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tracker := newProgressTracker(
			Options{Progress: func(ProgressEvent) {}}, []Spec{{Name: "tool"}},
			Config{Runs: len(runs), Interval: time.Millisecond},
		)
		for index := range runs {
			tracker.update(0, "tool", index+1, len(runs), false, false, runs[:index])
			tracker.update(0, "tool", index+1, len(runs), false, true, runs[:index+1])
		}
	}
}

func TestProgressSummariesMatchFinalStatistics(t *testing.T) {
	runs := []model.Run{
		{Index: 1, WallSeconds: 3, CPUUserSeconds: 1, MeanResidentBytes: 30,
			PeakPhysicalFootprintBytes: 300, PhysicalFootprintValid: true},
		{Index: 2, WallSeconds: 1, CPUUserSeconds: 2, MeanResidentBytes: 10,
			PeakPhysicalFootprintBytes: 100, PhysicalFootprintValid: true},
		{Index: 3, WallSeconds: 2, CPUUserSeconds: 3, MeanResidentBytes: 20,
			PeakPhysicalFootprintBytes: 200, PhysicalFootprintValid: true},
	}
	var estimates []model.Summary
	tracker := newProgressTracker(
		Options{Progress: func(event ProgressEvent) {
			if event.HasEstimate {
				estimates = append(estimates, event.Estimate)
			}
		}},
		[]Spec{{Name: "tool"}}, Config{Runs: len(runs), Interval: time.Millisecond},
	)
	for index := range runs {
		tracker.update(0, "tool", index+1, len(runs), false, true, runs[:index+1])
	}
	if len(estimates) != len(runs) {
		t.Fatalf("estimates = %d, want %d", len(estimates), len(runs))
	}
	for index, estimate := range estimates {
		want := model.Summarize(runs[:index+1])
		if !reflect.DeepEqual(estimate, want) {
			t.Fatalf("estimate %d = %+v, want %+v", index, estimate, want)
		}
	}
	if estimates[0].PeakPhysicalFootprintBytes.Mean != 300 {
		t.Fatal("earlier progress event was mutated by a later update")
	}
}
