package report

import (
	"fmt"
	"io"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/style"
)

func (renderer *Renderer) Write(writer io.Writer, format string) error {
	info, ok := LookupFormat(format)
	if !ok {
		return fmt.Errorf("unknown report format %q", format)
	}
	switch info.Name {
	case FormatText:
		return writeText(writer, renderer)
	case FormatJSON:
		return writeJSON(writer, renderer)
	case FormatMarkdown:
		return renderer.WriteMarkdownWithCharts(writer, nil, "")
	case FormatHTML:
		return writeHTML(writer, renderer)
	case FormatSVG:
		return writeSVG(writer, renderer)
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
