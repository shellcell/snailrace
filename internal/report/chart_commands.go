package report

import (
	"fmt"
	"html"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

func commandLegendChart(report model.Report) svgChart {
	const rowHeight = 48
	height := 50
	var body strings.Builder
	body.WriteString(svgChartStyle)
	fmt.Fprint(&body, `<rect width="100%" height="100%" rx="8" fill="#3b4252"/>`)
	fmt.Fprint(&body, `<text x="16" y="24" class="title">COMMAND LEGEND</text>`)
	for index, benchmark := range report.Benchmarks {
		lines := wrapText(fullCommand(benchmark), 76)
		y := height
		fmt.Fprintf(
			&body, `<text x="16" y="%d" class="label" style="fill:%s">%d. %s</text>`,
			y, toolColor(index), index+1,
			html.EscapeString(reportToolLabel(report, index)),
		)
		for _, line := range lines {
			y += 15
			fmt.Fprintf(
				&body, `<text x="34" y="%d" class="value">%s</text>`,
				y, html.EscapeString(line),
			)
		}
		height = y + rowHeight/2
	}
	return svgChart{
		kind: "commands", title: "Command legend", slug: "command-legend",
		description: commandLegendChartDescription(),
		body:        body.String(), height: height,
	}
}

func commandLegendChartDescription() string {
	return "Maps stable chart colors to the exact commands measured. This is a legend, " +
		"not a measurement chart."
}
