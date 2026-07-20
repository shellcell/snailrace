package runner

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shellcell/snailrace/internal/platform"
)

func TestRunOnceUsesInjectedOutputWriter(t *testing.T) {
	var output bytes.Buffer
	_, err := runOnce(
		context.Background(),
		Spec{Name: "output", Shell: "printf visible; printf error >&2"},
		time.Millisecond,
		Options{ShowOutput: true, Output: &output},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"visible", "error"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("injected output does not contain %q: %q", expected, output.String())
		}
	}
}

func TestRunOnceRecordsNonZeroExit(t *testing.T) {
	run, err := runOnce(
		context.Background(), Spec{Name: "exit", Shell: "exit 7"},
		time.Millisecond, Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if run.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", run.ExitCode)
	}
}

func TestMeanResidentUsesObservedTimeWeights(t *testing.T) {
	var samples sampleAggregate
	started := time.Unix(0, 0)
	samples.observe(platform.Metrics{ResidentBytes: 100}, started)
	samples.observe(platform.Metrics{ResidentBytes: 200}, started.Add(time.Second))
	samples.addCoverage(started.Add(2 * time.Second))
	if got := samples.meanResident(); got != 150 {
		t.Fatalf("weighted mean RSS = %g, want 150", got)
	}
}

func TestUpdatePeaksTracksPhysicalFootprint(t *testing.T) {
	var samples sampleAggregate
	samples.observe(platform.Metrics{
		PhysicalFootprintBytes: 100, PhysicalFootprintValid: true,
	}, time.Unix(0, 0))
	samples.observe(platform.Metrics{
		PhysicalFootprintBytes: 250, PhysicalFootprintValid: true,
	}, time.Unix(1, 0))
	samples.observe(platform.Metrics{
		PhysicalFootprintBytes: 200, PhysicalFootprintValid: true,
	}, time.Unix(2, 0))
	if samples.peak.PhysicalFootprintBytes != 250 {
		t.Fatalf("peak physical footprint = %d, want 250", samples.peak.PhysicalFootprintBytes)
	}
}

func TestUpdatePeaksInvalidatesPartialPhysicalFootprint(t *testing.T) {
	var samples sampleAggregate
	samples.observe(platform.Metrics{
		PhysicalFootprintBytes: 100, PhysicalFootprintValid: true,
	}, time.Unix(0, 0))
	samples.observe(
		platform.Metrics{PhysicalFootprintBytes: 200}, time.Unix(1, 0),
	)
	if samples.peak.PhysicalFootprintValid {
		t.Fatal("a run with an invalid physical-footprint sample should be unavailable")
	}
}

func TestMonitorUsesInjectedSampler(t *testing.T) {
	monitor := startMonitor(42, time.Hour, func(pid int) (platform.Metrics, bool) {
		if pid != 42 {
			t.Fatalf("sampled PID = %d, want 42", pid)
		}
		return platform.Metrics{ResidentBytes: 123, Processes: 1}, true
	})
	samples := monitor.finish()
	if samples.sampleCount != 1 || samples.peak.ResidentBytes != 123 {
		t.Fatalf("samples = %+v", samples)
	}
}

func TestMonitorPreservesInvalidPhysicalFootprintSample(t *testing.T) {
	monitor := startMonitor(42, time.Hour, func(int) (platform.Metrics, bool) {
		return platform.Metrics{PhysicalFootprintInvalid: true}, false
	})
	samples := monitor.finish()
	if samples.sampleCount != 0 || !samples.peak.PhysicalFootprintInvalid ||
		samples.peak.PhysicalFootprintValid {
		t.Fatalf("invalid sample = %+v", samples)
	}
}
