package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/style"
)

func Write(writer io.Writer, format string, report model.Report) error {
	if strings.ToLower(format) != "json" {
		report = safeDisplayReport(report)
	}
	switch strings.ToLower(format) {
	case "text", "txt":
		return writeText(writer, report)
	case "json":
		return writeJSON(writer, report)
	case "markdown", "md":
		return writeMarkdown(writer, report)
	case "html":
		return writeHTML(writer, report)
	case "svg":
		return writeSVG(writer, report)
	default:
		return fmt.Errorf("unknown report format %q", format)
	}
}

func safeDisplayReport(report model.Report) model.Report {
	report.Benchmarks = append([]model.Benchmark(nil), report.Benchmarks...)
	for index := range report.Benchmarks {
		report.Benchmarks[index].Tool.Name = style.SafeText(
			report.Benchmarks[index].Tool.Name,
		)
	}
	return report
}

func ValidFormat(format string) bool {
	switch strings.ToLower(format) {
	case "text", "txt", "json", "markdown", "md", "html", "svg":
		return true
	default:
		return false
	}
}
