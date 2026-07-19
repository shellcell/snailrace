package model

import "testing"

func TestSamplingReliableBoundaries(t *testing.T) {
	interval := 0.01
	tests := []struct {
		name string
		runs []Run
		want bool
	}{
		{"no runs", nil, false},
		{"reliable", []Run{{WallSeconds: 0.02, SampleCount: 2, SampleCoverageSeconds: 0.01}}, true},
		{"short", []Run{{WallSeconds: 0.019, SampleCount: 2, SampleCoverageSeconds: 0.01}}, false},
		{"few samples", []Run{{WallSeconds: 0.02, SampleCount: 1, SampleCoverageSeconds: 0.01}}, false},
		{"low coverage", []Run{{WallSeconds: 0.02, SampleCount: 2, SampleCoverageSeconds: 0.009}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			benchmark := Benchmark{Runs: test.runs}
			if got := SamplingReliable(benchmark, interval); got != test.want {
				t.Fatalf("reliable = %v, want %v", got, test.want)
			}
		})
	}
}
