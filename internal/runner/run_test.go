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
	metrics := platform.Metrics{
		ResidentByteSeconds: 300, SampleCoverageSeconds: 2,
		ResidentByteSamples: 999, SampleCount: 3,
	}
	if got := sampledMeanResident(metrics); got != 150 {
		t.Fatalf("weighted mean RSS = %g, want 150", got)
	}
}

func TestUpdatePeaksTracksPhysicalFootprint(t *testing.T) {
	peak := platform.Metrics{PhysicalFootprintBytes: 100}
	updatePeaks(&peak, platform.Metrics{PhysicalFootprintBytes: 250})
	updatePeaks(&peak, platform.Metrics{PhysicalFootprintBytes: 200})
	if peak.PhysicalFootprintBytes != 250 {
		t.Fatalf("peak physical footprint = %d, want 250", peak.PhysicalFootprintBytes)
	}
}
