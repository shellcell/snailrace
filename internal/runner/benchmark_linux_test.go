package runner

import (
	"context"
	"testing"
	"time"
)

func TestPinnedExecutionDoesNotLeakDescriptorsToChild(t *testing.T) {
	// The child inspects its own descriptor table with shell builtins: a pin
	// descriptor leaked through exec would appear at 3 or above. Sampling
	// PeakFileDescriptors instead is unreliable — libc startup transiently
	// opens files (locale archive, gconv), which a sample may legitimately
	// catch.
	benchmarks, err := Benchmark(
		context.Background(),
		[]Spec{{Name: "fds", Args: []string{
			"/bin/sh", "-c",
			"[ ! -e /proc/self/fd/3 ] && [ ! -e /proc/self/fd/9 ]",
		}}},
		Config{Runs: 1, Interval: time.Millisecond}, Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if code := benchmarks[0].Runs[0].ExitCode; code != 0 {
		t.Fatalf("child saw inherited descriptors (exit %d)", code)
	}
}
