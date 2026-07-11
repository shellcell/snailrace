package report

import (
	"fmt"
	"io"

	"perftool/internal/model"
)

func writeMarkdownCommandLegend(writer io.Writer, report model.Report) {
	fmt.Fprintln(writer, "## Commands")
	for index, benchmark := range report.Benchmarks {
		fmt.Fprintf(
			writer, "\n%d. **%s**\n\n```sh\n%s\n```\n",
			index+1, escapeMarkdown(reportToolLabel(report, index)),
			fullCommand(benchmark),
		)
	}
	fmt.Fprintln(writer)
}
