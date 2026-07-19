package runner

import (
	"context"
	"testing"
	"time"
)

func TestPinnedExecutableDoesNotInflateFileDescriptorCount(t *testing.T) {
	benchmarks, err := Benchmark(
		context.Background(),
		[]Spec{{Name: "sleep", Args: []string{"/bin/sleep", "0.05"}}},
		Config{Runs: 1, Interval: time.Millisecond}, Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := benchmarks[0].Runs[0].PeakFileDescriptors; got > 3 {
		t.Fatalf("peak file descriptors = %.0f, want no inherited pin descriptor", got)
	}
}
