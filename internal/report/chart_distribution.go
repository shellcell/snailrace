package report

import (
	"fmt"
	"html"
	"math"
	"strings"

	"perftool/internal/model"
)

func absoluteDistributionChart(report model.Report, metric chartMetric) svgChart {
	if !available(metricRows[metric.row], report.Host.OS) {
		return unavailableChart("RUN DISTRIBUTION · "+metric.name, report.Host.OS)
	}
	maximum := 0.0
	for _, benchmark := range report.Benchmarks {
		stats := metric.stats(benchmark)
		maximum = math.Max(maximum, math.Max(stats.Max, stats.CI95High))
	}
	if maximum <= 0 {
		maximum = 1
	}
	const left, plotWidth, rowHeight = 210, 390, 36
	height := 68 + len(report.Benchmarks)*rowHeight
	position := func(value float64) float64 {
		return left + math.Max(0, value)/maximum*plotWidth
	}
	var body strings.Builder
	body.WriteString(svgChartStyle)
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="24" class="title">RUN DISTRIBUTION · %s</text>`+
			`<text x="16" y="43" class="subtitle">dots = runs · diamond = mean · whisker = mean 95%% CI</text>`,
		html.EscapeString(metric.name),
	)
	fmt.Fprintf(
		&body,
		`<text x="%d" y="59" class="value">0</text>`+
			`<text x="%d" y="59" text-anchor="end" class="value">%s</text>`,
		left, left+plotWidth, metric.format(maximum),
	)
	baseline := baselineIndex(report)
	for index, benchmark := range report.Benchmarks {
		y := 76 + index*rowHeight
		color := chartColor(index, baseline)
		fmt.Fprintf(
			&body,
			`<text x="16" y="%d" class="label" style="fill:%s">%s</text>`+
				`<path d="M %d %d H %d" class="axis"/>`,
			y, color, html.EscapeString(clip(reportToolLabel(report, index), 28)),
			left, y-4, left+plotWidth,
		)
		for runIndex, run := range benchmark.Runs {
			jitter := (runIndex%3 - 1) * 5
			fmt.Fprintf(
				&body, `<circle cx="%.1f" cy="%d" r="3.5" fill="%s" opacity=".65"/>`,
				position(metric.run(run)), y-4+jitter, color,
			)
		}
		stats := metric.stats(benchmark)
		low, high := position(stats.CI95Low), position(stats.CI95High)
		mean := position(stats.Mean)
		fmt.Fprintf(
			&body,
			`<path d="M %.1f %d H %.1f M %.1f %d v 10 M %.1f %d v 10" `+
				`stroke="#eceff4" stroke-width="1.5"/>`+
				`<path d="M %.1f %d l 6 6 -6 6 -6 -6 z" fill="%s"/>`+
				`<text x="620" y="%d" class="value">%s</text>`,
			low, y-4, high, low, y-9, high, y-9,
			mean, y-10, color, y, metric.format(stats.Mean),
		)
	}
	return svgChart{
		kind: "distribution", title: metric.name, slug: chartSlug(metric.name),
		body: body.String(), height: height,
	}
}

func unavailableChart(title, operatingSystem string) svgChart {
	body := svgChartStyle +
		`<rect width="100%" height="100%" rx="8" fill="#3b4252"/>` +
		fmt.Sprintf(`<text x="16" y="24" class="title">%s</text>`, html.EscapeString(title)) +
		fmt.Sprintf(
			`<text x="16" y="48" class="subtitle">N/A on %s</text>`,
			html.EscapeString(operatingSystem),
		)
	return svgChart{
		kind: "distribution", title: title, slug: chartSlug(title),
		body: body, height: 66,
	}
}
