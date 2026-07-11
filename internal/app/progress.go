package app

import (
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"perftool/internal/runner"
)

func progressRenderer(writer io.Writer, enabled bool) func(runner.ProgressEvent) {
	file, ok := writer.(interface{ Fd() uintptr })
	if !enabled || !ok || !term.IsTerminal(int(file.Fd())) {
		return nil
	}
	var mutex sync.Mutex
	renderedLines := 0
	cursorHidden := false
	return func(event runner.ProgressEvent) {
		mutex.Lock()
		defer mutex.Unlock()
		if event.Finished {
			if cursorHidden {
				fmt.Fprint(writer, "\x1b[?25h\r\n")
			}
			renderedLines = 0
			return
		}
		if !cursorHidden {
			fmt.Fprint(writer, "\x1b[?25l")
			cursorHidden = true
		}
		clearProgress(writer, renderedLines)
		width, _, _ := term.GetSize(int(file.Fd()))
		lines := append(
			[]string{progressStatus(event)}, progressRaceLines(event, width)...,
		)
		fmt.Fprint(writer, strings.Join(lines, "\n"))
		renderedLines = len(lines)
	}
}

func progressStatus(event runner.ProgressEvent) string {
	phase := fmt.Sprintf("run %d/%d", event.Iteration, event.Iterations)
	if event.Warmup {
		phase = fmt.Sprintf("warmup %d/%d", event.Iteration, event.Iterations)
	}
	estimate := "calibrating estimates"
	if event.FixedDuration > 0 {
		estimate = "expected wall " + progressDuration(event.FixedDuration.Seconds()) +
			" · CPU/RSS calibrating"
	}
	if event.HasEstimate {
		wall := event.Estimate.WallSeconds.Mean
		if event.FixedDuration > 0 {
			wall = event.FixedDuration.Seconds()
		}
		estimate = fmt.Sprintf(
			"expected wall %s · CPU %s · RSS %s",
			progressDuration(wall),
			progressDuration(event.Estimate.CPUTotalSeconds.Mean),
			progressBytes(event.Estimate.PeakResidentBytes.Mean),
		)
	}
	step := event.Completed + 1
	if step > event.Total {
		step = event.Total
	}
	return fmt.Sprintf(
		"\x1b[38;2;136;192;208m[%d/%d]\x1b[0m "+
			"🐌 tool %d/%d %s · %s · %s · ETA %s",
		step, event.Total, event.Tool, event.ToolCount, event.ToolName, phase,
		estimate, progressClock(event.ETA),
	)
}

func clearProgress(writer io.Writer, lines int) {
	if lines == 0 {
		return
	}
	fmt.Fprint(writer, "\r\x1b[2K")
	for line := 1; line < lines; line++ {
		fmt.Fprint(writer, "\x1b[1A\r\x1b[2K")
	}
}

func progressDuration(seconds float64) string {
	if seconds < 0.001 {
		return fmt.Sprintf("%.3g us", seconds*1e6)
	}
	if seconds < 1 {
		return fmt.Sprintf("%.3g ms", seconds*1e3)
	}
	return fmt.Sprintf("%.3g s", seconds)
}

func progressBytes(value float64) string {
	units := [...]string{"B", "KiB", "MiB", "GiB"}
	unit := 0
	for math.Abs(value) >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.3g %s", value, units[unit])
}

func progressClock(duration time.Duration) string {
	if duration <= 0 {
		return "--"
	}
	if duration < time.Second {
		return duration.Round(10 * time.Millisecond).String()
	}
	return duration.Round(time.Second).String()
}
