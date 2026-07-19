package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/shellcell/snailrace/internal/model"
)

func reportPath(directory, format string, report model.Report) string {
	return reportPathWithStem(directory, format, reportStem(report))
}

func reportPathWithStem(directory, format, stem string) string {
	extension := strings.ToLower(format)
	switch extension {
	case "text":
		extension = "txt"
	case "markdown":
		extension = "md"
	}
	return filepath.Join(directory, stem+"."+extension)
}

func reportStem(report model.Report) string {
	tool := reportLabel(report.Benchmarks)
	timestamp := report.MeasuredAt.Format("20060102-150405")
	return fmt.Sprintf("snail-%s-%s", tool, timestamp)
}

func reportLabel(benchmarks []model.Benchmark) string {
	if len(benchmarks) == 0 {
		return "unknown"
	}
	if len(benchmarks) == 1 {
		return slug(benchmarks[0].Tool.Name)
	}
	label := slug(benchmarks[0].Tool.Name) + "-vs-" +
		slug(benchmarks[1].Tool.Name)
	if len(benchmarks) > 2 {
		label += fmt.Sprintf("-and-%d", len(benchmarks)-2)
	}
	return label
}

func slug(value string) string {
	var result strings.Builder
	lastHyphen := false
	for _, character := range strings.ToLower(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			result.WriteRune(character)
			lastHyphen = false
		} else if !lastHyphen && result.Len() > 0 {
			result.WriteByte('-')
			lastHyphen = true
		}
		if result.Len() >= 32 {
			break
		}
	}
	value = strings.Trim(result.String(), "-")
	if value == "" {
		return "command"
	}
	return value
}
