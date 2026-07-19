package report

import (
	"fmt"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

func writeTextCommandLegend(writer io.Writer, report model.Report) {
	fmt.Fprintln(writer, "Commands\t")
	for index, benchmark := range report.Benchmarks {
		label := fmt.Sprintf("%d. %s", index+1, reportToolLabel(report, index))
		for lineIndex, line := range wrapText(fullCommand(benchmark), 100) {
			if lineIndex == 0 {
				fmt.Fprintf(writer, "%s\t%s\n", label, line)
			} else {
				fmt.Fprintf(writer, "\t%s\n", line)
			}
		}
	}
	fmt.Fprintln(writer)
}
