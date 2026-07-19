package app

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shellcell/snailrace/internal/platform"
)

func platformNotes(
	tui bool,
	duration time.Duration,
	comparison, automaticBaseline bool,
	orderSeed int64,
	outputMode string,
) []string {
	notes := []string{
		"Sample standard deviation uses n-1; the mean interval uses Student's t.",
		"Quantiles use R-7 linear interpolation; no outliers are removed.",
		"Sampled process-tree peaks may miss processes shorter than the interval.",
		"CPU time and maxrss use operating-system accounting returned by wait.",
		platform.MaxRSSDescription(),
		"Tree RSS is a sampled sum and may double-count shared pages.",
		"Linked footprint excludes runtime-loaded plugins and child executables.",
		"Commands that daemonize or escape their process group are unsupported.",
		"Command output mode: " + outputMode + ".",
		fmt.Sprintf(
			"Execution used randomized counterbalanced blocks (seed %d); final partial blocks are not fully balanced.",
			orderSeed,
		),
	}
	if runtime.GOOS == "darwin" {
		notes = append(
			notes,
			"File descriptor metrics are unavailable on macOS.",
			"Physical footprint is the sampled sum charged to the process group.",
			"Dyld shared-cache libraries have no standalone size and are excluded from disk footprint.",
		)
	}
	if tui && duration > 0 {
		notes = append(
			notes,
			"TUI runs use one launch-to-exit window; configured duration excludes teardown.",
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
			"Balanced ranking is descriptive and uses point estimates, not uncertainty.",
			"Balanced ranking omits categories with nonpositive or unavailable values.",
			"RAM aggregate is the geometric mean of mean and peak resident memory.",
			"Paired 95% intervals are pointwise and unadjusted for multiple comparisons.",
			"Statistical direction does not imply practical significance.",
			"Resource better/worse labels assume commands perform equivalent work.",
			"Sampled metrics require two intervals, two valid samples, and observed coverage.",
		)
		if automaticBaseline {
			notes = append(
				notes,
				"The baseline was selected from these data; comparisons are exploratory post-selection.",
			)
		}
	}
	return notes
}
