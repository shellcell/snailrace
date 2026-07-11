package app

import (
	"runtime"
	"time"
)

func platformNotes(tui bool, duration time.Duration, comparison bool) []string {
	notes := []string{
		"Sample standard deviation uses n-1; the mean interval uses Student's t.",
		"Quantiles use R-7 linear interpolation; no outliers are removed.",
		"Sampled process-tree peaks may miss processes shorter than the interval.",
		"CPU time and fallback peak RSS use operating-system child resource usage.",
		"Linked footprint excludes runtime-loaded plugins and child executables.",
	}
	if runtime.GOOS == "darwin" {
		notes = append(
			notes,
			"File descriptor metrics are unavailable on macOS.",
			"macOS process sampling invokes ps after the first interval.",
		)
	}
	if tui && duration > 0 {
		notes = append(
			notes,
			"TUI fixed-duration wall time ends at termination; CPU includes teardown.",
		)
	} else if tui {
		notes = append(
			notes,
			"Unlimited interactive results describe the observed user session.",
		)
	}
	if comparison {
		notes = append(
			notes,
			"Balanced ranking equally weights normalized time, CPU, RAM, and linked size.",
			"RAM aggregate is the geometric mean of mean and peak resident memory.",
			"Δ percentages compare means; better/worse requires a paired 95% interval.",
			"Resource better/worse labels assume commands perform equivalent work.",
			"Sampled metrics require every compared run to span at least two intervals.",
		)
	}
	return notes
}
