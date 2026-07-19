package app

import (
	"bytes"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/runner"
)

var progressColorSequence = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestProgressRaceAdvancesBetterExpectedTool(t *testing.T) {
	event := runner.ProgressEvent{
		Completed: 2, Total: 4, Elapsed: time.Second, ETA: time.Second,
		Estimates: []runner.ProgressEstimate{
			progressEstimate("better", 1),
			progressEstimate("worse", 2),
		},
	}
	lines := progressRaceLines(event, 80)
	if len(lines) != 2 {
		t.Fatalf("race lines = %d, want 2", len(lines))
	}
	for _, line := range lines {
		clean := progressColorSequence.ReplaceAllString(line, "")
		if !strings.HasSuffix(clean, "🏁") {
			t.Fatalf("unfinished track does not end in a finish flag: %q", line)
		}
		if !strings.Contains(clean, "🐌") {
			t.Fatalf("track has no snail: %q", line)
		}
	}
	if progressTrailLength(lines[0]) <= progressTrailLength(lines[1]) {
		t.Fatalf("better tool should be farther ahead:\n%s\n%s", lines[0], lines[1])
	}
	if !strings.Contains(lines[0], "▁") || !strings.Contains(lines[1], "▅") {
		t.Fatalf("higher RAM should leave a thicker trail:\n%s\n%s", lines[0], lines[1])
	}
	colorPrefix := strings.TrimSuffix(progressToolColor(0, ""), "\x1b[0m")
	if !strings.Contains(lines[0], colorPrefix+"better") {
		t.Fatal("tool name does not use its report color")
	}
}

func TestProgressWinnerErasesFinishFlag(t *testing.T) {
	winner := progressEstimate("winner", 1)
	winner.Completed, winner.Total = 4, 4
	loser := progressEstimate("loser", 4)
	loser.Completed, loser.Total = 4, 4
	event := runner.ProgressEvent{
		Completed: 8, Total: 8,
		Estimates: []runner.ProgressEstimate{winner, loser},
	}
	lines := progressRaceLines(event, 80)
	win := progressColorSequence.ReplaceAllString(lines[0], "")
	lose := progressColorSequence.ReplaceAllString(lines[1], "")
	if strings.Contains(win, "🏁") || !strings.HasSuffix(win, "🐌") {
		t.Fatalf("winner should reach the finish and erase its flag: %q", win)
	}
	if !strings.HasSuffix(lose, "🏁") {
		t.Fatalf("loser should keep its finish flag standing: %q", lose)
	}
}

func TestProgressTrackUsesThreeQuartersForMeasurements(t *testing.T) {
	before := progressTrackFraction(4, 10, 0.5)
	after := progressTrackFraction(5, 10, 0.5)
	if difference := after - before; math.Abs(difference-0.075) > 1e-9 {
		t.Fatalf("one of ten measurements advanced %.3f, want 0.075", difference)
	}
	if got := progressTrackFraction(10, 10, 1); got != 1 {
		t.Fatalf("completed winning track = %g, want 1", got)
	}
}

func TestProgressTrailProvidesMultipleRAMGradations(t *testing.T) {
	tools := make([]runner.ProgressEstimate, 0, 6)
	for _, scale := range []float64{1, 1.25, 1.5, 2, 3, 4} {
		tools = append(tools, progressEstimate("tool", scale))
	}
	unique := make(map[string]bool)
	for _, trail := range progressTrailCharacters(tools) {
		unique[trail] = true
	}
	if len(unique) < 5 {
		t.Fatalf("RAM trail gradations = %d, want at least 5", len(unique))
	}
}

func TestProgressColorsToolsBeforeCalibration(t *testing.T) {
	event := runner.ProgressEvent{Estimates: []runner.ProgressEstimate{
		{ToolName: "first", Total: 10}, {ToolName: "second", Total: 10},
	}}
	for index, line := range progressRaceLines(event, 80) {
		prefix := strings.TrimSuffix(progressToolColor(index, ""), "\x1b[0m")
		if !strings.Contains(line, prefix+event.Estimates[index].ToolName) ||
			!strings.Contains(line, prefix+"🐌") {
			t.Fatalf("uncalibrated tool %d is not consistently colored: %q", index, line)
		}
	}
}

func TestProgressStatusRemainsFirstLineContent(t *testing.T) {
	status := progressStatus(runner.ProgressEvent{
		ToolName: "first", Tool: 1, ToolCount: 2,
		Iteration: 2, Iterations: 3, Completed: 1, Total: 6,
		ETA: time.Second,
	}, 0)
	for _, expected := range []string{"[2/6]", "🐌 ETA 1s", "tool 1/2 first", "run 2/3"} {
		if !strings.Contains(status, expected) {
			t.Fatalf("status does not contain %q: %q", expected, status)
		}
	}
}

func TestProgressStatusAlignsNumbersAndName(t *testing.T) {
	status := progressStatus(runner.ProgressEvent{
		ToolName: "a", Tool: 2, ToolCount: 4,
		Iteration: 3, Iterations: 300, Completed: 0, Total: 1204,
	}, 8)
	expected := []string{
		"[   1/1204]",                       // step padded to the total width
		"tool 2/4",                          // tool index within one-digit count
		"a" + strings.Repeat(" ", 7) + " ·", // name padded to the longest name
		"run   3/300",                       // iteration padded to the run count
	}
	for _, want := range expected {
		if !strings.Contains(status, want) {
			t.Fatalf("status not aligned, missing %q: %q", want, status)
		}
	}
}

func TestProgressRenderedRowsCountsWrappedLines(t *testing.T) {
	lines := []string{"12345678901", "short"}
	if got := progressRenderedRows(lines, 10); got != 3 {
		t.Fatalf("rendered rows = %d, want 3", got)
	}
}

func TestProgressRenderedRowsIgnoresANSIAndCountsKnownEmojiWidth(t *testing.T) {
	line := "\x1b[38;2;136;192;208m🐌\x1b[0m123456789"
	if got := progressRenderedRows([]string{line}, 10); got != 2 {
		t.Fatalf("rendered rows = %d, want 2", got)
	}
}

func TestProgressRendererRecalculatesRowsAfterResize(t *testing.T) {
	primary, terminal, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	defer terminal.Close()
	if err := pty.Setsize(terminal, &pty.Winsize{Cols: 200, Rows: 24}); err != nil {
		t.Fatal(err)
	}
	output := progressTerminalBuffer{fd: terminal.Fd()}
	render := progressRenderer(&output, true, 0)
	if render == nil {
		t.Fatal("progress renderer disabled for PTY")
	}
	event := runner.ProgressEvent{
		ToolName: "first", Tool: 1, ToolCount: 1,
		Iteration: 1, Iterations: 10, Total: 10,
	}
	render(event)
	redrawStart := output.Len()
	if err := pty.Setsize(terminal, &pty.Winsize{Cols: 40, Rows: 24}); err != nil {
		t.Fatal(err)
	}
	render(event)

	previous := []string{progressStatus(event, 0)}
	wantRows := progressRenderedRows(previous, 40)
	redraw := output.String()[redrawStart:]
	if got := strings.Count(redraw, "\x1b[1A") + 1; got != wantRows {
		t.Fatalf("cleared rows after resize = %d, want %d", got, wantRows)
	}
}

type progressTerminalBuffer struct {
	bytes.Buffer
	fd uintptr
}

func (buffer *progressTerminalBuffer) Fd() uintptr { return buffer.fd }

func progressEstimate(name string, scale float64) runner.ProgressEstimate {
	stats := model.Stats{Mean: scale}
	return runner.ProgressEstimate{
		ToolName: name, Completed: 1, Total: 2, HasEstimate: true,
		Runs: []model.Run{{
			WallSeconds: scale, CPUUserSeconds: scale,
			MeanResidentBytes: scale, PeakResidentBytes: scale,
		}},
		DiskFootprintBytes: int64(scale * 100),
		Estimate: model.Summary{
			WallSeconds: stats, CPUTotalSeconds: stats,
			MeanResidentBytes: stats, PeakResidentBytes: stats,
		},
	}
}

func progressTrailLength(line string) int {
	length := 0
	for _, glyph := range progressTrailGlyphs {
		length += strings.Count(line, glyph)
	}
	return length
}
