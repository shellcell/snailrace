package runner

import (
	"context"
	"testing"
	"time"
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
