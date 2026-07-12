package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBenchmarkInterruptionKeepsCompletedRounds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	specs := []Spec{
		{Name: "a", Args: []string{"/bin/sleep", "0.002"}},
		{Name: "b", Args: []string{"/bin/sleep", "0.002"}},
	}
	// Cancel as soon as the first full round completes; the unfinished second
	// round should be discarded, leaving one run per tool.
	progress := func(event ProgressEvent) {
		if completedRoundCount(event.Estimates) >= 1 {
			cancel()
		}
	}
	benchmarks, err := Benchmark(
		ctx, specs,
		Config{Runs: 3, Interval: time.Millisecond},
		Options{Progress: progress},
	)
	if !errors.Is(err, ErrInterrupted) {
		t.Fatalf("err = %v, want ErrInterrupted", err)
	}
	for index, benchmark := range benchmarks {
		if len(benchmark.Runs) != 1 {
			t.Fatalf("tool %d recorded %d runs, want 1 completed round", index, len(benchmark.Runs))
		}
	}
}

func TestBenchmarkInterruptionBeforeAnyRoundFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Benchmark(
		ctx,
		[]Spec{{Name: "a", Args: []string{"/bin/sleep", "0.002"}}},
		Config{Runs: 3, Interval: time.Millisecond},
		Options{},
	)
	if errors.Is(err, ErrInterrupted) || err == nil {
		t.Fatalf("err = %v, want a plain cancellation error with no partial results", err)
	}
}

func completedRoundCount(estimates []ProgressEstimate) int {
	if len(estimates) == 0 {
		return 0
	}
	minimum := estimates[0].Completed
	for _, estimate := range estimates[1:] {
		if estimate.Completed < minimum {
			minimum = estimate.Completed
		}
	}
	return minimum
}
