package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/style"
)

func Write(writer io.Writer, format string, report model.Report) error {
	return NewRenderer(report).Write(writer, format)
}

func (renderer *Renderer) Write(writer io.Writer, format string) error {
	switch strings.ToLower(format) {
	case "text", "txt":
		return writeTextRenderer(writer, renderer)
	case "json":
		return writeJSONRenderer(writer, renderer)
	case "markdown", "md":
		return renderer.WriteMarkdownWithCharts(writer, nil, "")
	case "html":
		return writeHTMLRenderer(writer, renderer)
	case "svg":
		return writeSVGRenderer(writer, renderer)
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
