package report

import (
	"fmt"
	"html"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

func svgCommandLegend(output *strings.Builder, report model.Report, y int) int {
	fmt.Fprintf(output, `<text x="24" y="%d" class="h2">COMMANDS</text>`, y)
	y += 25
	for index, benchmark := range report.Benchmarks {
		fmt.Fprintf(
			output, `<text x="24" y="%d" class="h3" style="fill:%s">%d. %s</text>`,
			y, toolColor(index), index+1,
			html.EscapeString(clip(reportToolLabel(report, index), 80)),
		)
		y += 18
		for _, line := range wrapText(fullCommand(benchmark), 175) {
			fmt.Fprintf(
				output, `<text x="40" y="%d" class="muted">%s</text>`,
				y, html.EscapeString(line),
			)
			y += 17
		}
		y += 6
	}
	return y + 12
}
