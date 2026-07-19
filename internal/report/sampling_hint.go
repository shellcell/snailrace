package report

import (
	"fmt"

	"github.com/shellcell/snailrace/internal/analysis"
	"github.com/shellcell/snailrace/internal/model"
)

// SamplingLimitNote returns guidance when RAM is present but sampling-limited,
// or "" otherwise. It points at the current interval and advises lowering it
// rather than prescribing a specific value, since reliability also depends on
// run length and sample coverage. Safe to append to a report's notes.
func SamplingLimitNote(config model.Config, benchmarks []model.Benchmark) string {
	numeric := analysis.Calculate(config, benchmarks)
	if !(numeric.RAMPresent && !numeric.RAMAvailable) {
		return ""
	}
	return fmt.Sprintf(
		"RAM aggregate is sampling-limited: runs are short relative to the current %s ms "+
			"sampling interval. Re-run with a smaller -interval for reliable RAM.",
		formatNumber(config.IntervalMS),
	)
}

// currentIntervalLabel formats the configured sampling interval for hints.
func currentIntervalLabel(config model.Config) string {
	if config.IntervalMS <= 0 {
		return ""
	}
	return formatNumber(config.IntervalMS) + " ms"
}
