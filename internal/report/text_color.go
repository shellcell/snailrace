package report

import (
	"fmt"
	"sort"
	"strings"

	"perftool/internal/model"
	"perftool/internal/style"
)

func colorizeText(value string, report model.Report) string {
	value = colorComparisonValues(value)
	value = colorRankingValues(value)
	value = colorToolNames(value, report)
	replacements := []struct{ old, new string }{
		{" [BASELINE]", " " + ansi("136;192;208", "[BASELINE]")},
		{" · better", " · " + ansi("163;190;140", "better")},
		{" · worse", " · " + ansi("191;97;106", "worse")},
		{" · inconclusive", " · " + ansi("235;203;139", "inconclusive")},
		{" · sampling-limited", " · " + ansi("235;203;139", "sampling-limited")},
		{" · higher", " · " + ansi("129;161;193", "higher")},
		{" · lower", " · " + ansi("129;161;193", "lower")},
	}
	for _, replacement := range replacements {
		value = strings.ReplaceAll(value, replacement.old, replacement.new)
	}
	return value
}

func colorComparisonValues(value string) string {
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		color := ""
		switch {
		case strings.Contains(line, " · better"):
			color = "163;190;140"
		case strings.Contains(line, " · worse"):
			color = "191;97;106"
		case strings.Contains(line, " · inconclusive"),
			strings.Contains(line, " · sampling-limited"):
			color = "235;203;139"
		case strings.Contains(line, " · same"):
			color = "129;161;193"
		}
		if color == "" {
			continue
		}
		gaps := columnGaps(line)
		if len(gaps) < 3 {
			continue
		}
		baselineStart, baselineEnd := gaps[0][1], gaps[1][0]
		candidateStart, candidateEnd := gaps[1][1], gaps[2][0]
		lines[index] = line[:baselineStart] +
			ansi("129;161;193", line[baselineStart:baselineEnd]) +
			line[baselineEnd:candidateStart] +
			ansi(color, line[candidateStart:candidateEnd]) + line[candidateEnd:]
	}
	return strings.Join(lines, "\n")
}

func colorRankingValues(value string) string {
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		gaps := columnGaps(line)
		for gapIndex := len(gaps) - 1; gapIndex >= 1; gapIndex-- {
			start := gaps[gapIndex-1][1]
			end := gaps[gapIndex][0]
			segment := line[start:end]
			color := "191;97;106"
			if strings.Contains(segment, "(#1)") {
				color = "163;190;140"
			}
			line = line[:start] + ansi(color, segment) + line[end:]
		}
		lines[index] = line
	}
	return strings.Join(lines, "\n")
}

func columnGaps(value string) [][2]int {
	var result [][2]int
	for index := 0; index < len(value); {
		if value[index] != ' ' {
			index++
			continue
		}
		start := index
		for index < len(value) && value[index] == ' ' {
			index++
		}
		if index-start >= 2 {
			result = append(result, [2]int{start, index})
		}
	}
	return result
}

func colorToolNames(value string, report model.Report) string {
	type namedColor struct{ name, color string }
	items := make([]namedColor, 0, len(report.Benchmarks))
	for index, benchmark := range report.Benchmarks {
		items = append(items, namedColor{
			name:  benchmark.Tool.Name,
			color: terminalChartColor(index),
		})
	}
	sort.Slice(items, func(i, j int) bool { return len(items[i].name) > len(items[j].name) })
	for _, item := range items {
		value = strings.ReplaceAll(value, item.name, ansi(item.color, item.name))
	}
	return value
}

func terminalChartColor(index int) string {
	color := style.Tool(index)
	return fmt.Sprintf("%d;%d;%d", color.R, color.G, color.B)
}

func ansi(color, value string) string {
	return "\x1b[38;2;" + color + "m" + value + "\x1b[0m"
}
