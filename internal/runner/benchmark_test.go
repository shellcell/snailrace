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

func TestBenchmarkContinuesAfterNonZeroExit(t *testing.T) {
	benchmarks, err := Benchmark(
		context.Background(),
		[]Spec{
			{Name: "exit", Shell: "exit 7"},
			{Name: "true", Args: []string{"/bin/true"}},
		},
		Config{Runs: 3, Warmups: 1, Interval: time.Millisecond},
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	for index, benchmark := range benchmarks {
		if len(benchmark.Runs) != 3 {
			t.Fatalf("tool %d recorded %d runs, want 3", index, len(benchmark.Runs))
		}
	}
	for _, run := range benchmarks[0].Runs {
		if run.ExitCode != 7 {
			t.Fatalf("exit code = %d, want 7", run.ExitCode)
		}
	}
}

func TestBenchmarkRejectsInvalidInputs(t *testing.T) {
	validSpec := Spec{Name: "true", Args: []string{"/bin/true"}}
	validConfig := Config{Runs: 1, Interval: time.Millisecond}
	tests := []struct {
		name   string
		specs  []Spec
		config Config
	}{
		{"no specs", nil, validConfig},
		{"empty spec", []Spec{{Name: "empty"}}, validConfig},
		{"both command forms", []Spec{{Name: "both", Shell: "true", Args: []string{"true"}}}, validConfig},
		{"zero runs", []Spec{validSpec}, Config{Interval: time.Millisecond}},
		{"zero interval", []Spec{validSpec}, Config{Runs: 1}},
		{"bad schedule index", []Spec{validSpec}, Config{
			Runs: 1, Interval: time.Millisecond, MeasurementOrder: [][]int{{1}},
		}},
		{"wrong schedule rounds", []Spec{validSpec}, Config{
			Runs: 2, Interval: time.Millisecond, MeasurementOrder: [][]int{{0}},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Benchmark(
				context.Background(), test.specs, test.config, Options{},
			); err == nil {
				t.Fatal("expected validation error")
			}
		})
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
