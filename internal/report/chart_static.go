package report

import (
	"fmt"
	"html"
	"math"
	"strings"

	"perftool/internal/model"
)

func staticCostChart(
	report model.Report,
	name string,
	pick func(model.Benchmark) float64,
	format func(float64) string,
) svgChart {
	maximum := 0.0
	for _, benchmark := range report.Benchmarks {
		value := pick(benchmark)
		maximum = math.Max(maximum, value)
	}
	if maximum <= 0 {
		maximum = 1
	}
	const left, plotWidth, rowHeight = 210, 390, 34
	height := 64 + len(report.Benchmarks)*rowHeight
	var body strings.Builder
	body.WriteString(svgChartStyle)
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="24" class="title">%s</text>`+
			`<text x="16" y="42" class="subtitle">exact executable + linked-library bytes</text>`,
		html.EscapeString(name),
	)
	fmt.Fprintf(
		&body,
		`<text x="%d" y="55" class="value">0</text>`+
			`<text x="%d" y="55" text-anchor="end" class="value">%s</text>`,
		left, left+plotWidth, format(maximum),
	)
	baseline := baselineIndex(report)
	for index, benchmark := range report.Benchmarks {
		y := 72 + index*rowHeight
		value := pick(benchmark)
		x := left + value/maximum*plotWidth
		color := chartColor(index, baseline)
		fmt.Fprintf(
			&body,
			`<text x="16" y="%d" class="label" style="fill:%s">%s</text>`+
				`<path d="M %d %d H %d" class="axis"/>`+
				`<path d="M %.1f %d l 6 6 -6 6 -6 -6 z" fill="%s"/>`+
				`<text x="620" y="%d" class="value">%s</text>`,
			y, color, html.EscapeString(clip(reportToolLabel(report, index), 28)),
			left, y-4, left+plotWidth, x, y-10, color, y, format(value),
		)
	}
	return svgChart{
		kind: "distribution", title: name, slug: chartSlug(name),
		body: body.String(), height: height,
	}
}
