package app

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"perftool/internal/analysis"
	"perftool/internal/model"
	"perftool/internal/runner"
)

var progressTrailGlyphs = [...]string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

func progressRaceLines(event runner.ProgressEvent, terminalWidth int) []string {
	scores := progressScores(event)
	trails := progressTrailCharacters(event.Estimates)
	best := math.Inf(1)
	for _, score := range scores {
		if score > 0 && score < best {
			best = score
		}
	}
	if terminalWidth <= 0 {
		terminalWidth = 80
	}
	trackWidth := terminalWidth - 43
	if trackWidth < 8 {
		trackWidth = 8
	} else if trackWidth > 36 {
		trackWidth = 36
	}
	lines := make([]string, 0, len(event.Estimates))
	for index, tool := range event.Estimates {
		quality, scoreLabel := 0.0, "calibrating"
		if index < len(scores) && scores[index] > 0 && !math.IsInf(scores[index], 1) {
			quality = best / scores[index]
			scoreLabel = fmt.Sprintf("%.3fx", scores[index])
		}
		fraction := progressTrackFraction(tool.Completed, tool.Total, quality)
		distance := int(float64(trackWidth) * fraction)
		name := fmt.Sprintf("%-16s", progressToolName(tool.ToolName, 16))
		track := progressTrack(trails[index], distance, trackWidth)
		line := fmt.Sprintf(
			"  %s %3d/%-3d %11s %s",
			progressToolColor(index, name), tool.Completed, tool.Total,
			scoreLabel, progressToolColor(index, track),
		)
		lines = append(lines, line)
	}
	return lines
}

// progressTrack draws a snail on its lane with a finish flag planted at the
// end. Every lane shows its flag until the snail reaches it: only a snail that
// runs the full track (the winner, at fraction 1) knocks its flag down.
func progressTrack(trail string, distance, width int) string {
	if distance >= width {
		return strings.Repeat(trail, width) + "🐌"
	}
	gap := strings.Repeat(" ", width-distance-1)
	return strings.Repeat(trail, distance) + "🐌" + gap + "🏁"
}

func progressTrackFraction(completed, total int, quality float64) float64 {
	measurement := 0.0
	if total > 0 {
		measurement = float64(completed) / float64(total)
	}
	measurement = math.Max(0, math.Min(1, measurement))
	quality = math.Max(0, math.Min(1, quality))
	return 0.75*measurement + 0.25*quality
}

func progressScores(event runner.ProgressEvent) []float64 {
	scores := make([]float64, len(event.Estimates))
	benchmarks := make([]model.Benchmark, 0, len(event.Estimates))
	positions := make([]int, 0, len(event.Estimates))
	for index, tool := range event.Estimates {
		if !tool.HasEstimate {
			continue
		}
		benchmarks = append(benchmarks, model.Benchmark{
			Tool: model.ToolInfo{
				Name: tool.ToolName, DiskFootprintBytes: tool.DiskFootprintBytes,
			},
			Runs: tool.Runs, Summary: tool.Estimate,
		})
		positions = append(positions, index)
	}
	config := model.Config{Mode: "command", IntervalMS: event.IntervalMS}
	if event.FixedDuration > 0 {
		config.Mode = "tui"
		config.DurationSeconds = event.FixedDuration.Seconds()
	}
	for index, score := range analysis.BalancedIndexes(config, benchmarks) {
		scores[positions[index]] = score
	}
	return scores
}

func progressTrailCharacters(tools []runner.ProgressEstimate) []string {
	result := make([]string, len(tools))
	minimum := math.Inf(1)
	values := make([]float64, len(tools))
	for index, tool := range tools {
		if !tool.HasEstimate {
			continue
		}
		mean := tool.Estimate.MeanResidentBytes.Mean
		peak := tool.Estimate.PeakResidentBytes.Mean
		if mean > 0 && peak > 0 {
			values[index] = math.Sqrt(mean * peak)
			minimum = math.Min(minimum, values[index])
		}
	}
	for index, value := range values {
		level := 0
		if value <= 0 || math.IsInf(minimum, 1) {
			result[index] = progressTrailGlyphs[level]
			continue
		}
		ratio := value / minimum
		relativeLevel := int(math.Round(math.Log2(ratio) / 2 * 7))
		absoluteLevel := int(math.Floor(math.Log2(value / (16 * 1024 * 1024))))
		level = max(relativeLevel, absoluteLevel)
		if level < 0 {
			level = 0
		} else if level >= len(progressTrailGlyphs) {
			level = len(progressTrailGlyphs) - 1
		}
		result[index] = progressTrailGlyphs[level]
	}
	return result
}

func progressToolName(name string, maximum int) string {
	if utf8.RuneCountInString(name) <= maximum {
		return name
	}
	runes := []rune(name)
	return string(runes[:maximum-3]) + "..."
}
