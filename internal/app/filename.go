package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/report"
)

func reportPathWithStem(directory, format, stem string) string {
	extension := strings.ToLower(format)
	if info, ok := report.LookupFormat(format); ok {
		extension = info.Extension
	}
	return filepath.Join(directory, stem+"."+extension)
}

func reportStem(report model.Report) string {
	names := make([]string, len(report.Benchmarks))
	for index, benchmark := range report.Benchmarks {
		names[index] = benchmark.Tool.Name
	}
	return reportStemFor(report.MeasuredAt, names)
}

func reportStemFor(measuredAt time.Time, toolNames []string) string {
	tool := reportLabelNames(toolNames)
	timestamp := measuredAt.Format("20060102-150405")
	return fmt.Sprintf("snail-%s-%s", tool, timestamp)
}

func reportLabelNames(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}
	if len(names) == 1 {
		return slug(names[0])
	}
	label := slug(names[0]) + "-vs-" + slug(names[1])
	if len(names) > 2 {
		label += fmt.Sprintf("-and-%d", len(names)-2)
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
