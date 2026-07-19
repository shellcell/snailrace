package report

import (
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

func absoluteDistributionChart(report model.Report, metric chartMetric) svgChart {
	if !available(metricRows[metric.row], report.Host.OS) {
		return unavailableChart("RUN DISTRIBUTION · "+metric.name, report.Host.OS)
	}
	minimum, maximum := 0.0, 0.0
	for _, benchmark := range report.Benchmarks {
		stats := metric.stats(benchmark)
		maximum = math.Max(maximum, stats.Max)
		if stats.CI95Valid {
			minimum = math.Min(minimum, stats.CI95Low)
			maximum = math.Max(maximum, stats.CI95High)
		}
	}
	if maximum <= minimum {
		maximum = 1
	}
	const left, plotWidth, rowHeight = 210, 390, 36
	height := 68 + len(report.Benchmarks)*rowHeight
	position := func(value float64) float64 {
		return left + (value-minimum)/(maximum-minimum)*plotWidth
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
		`<text x="%d" y="59" class="value">%s</text>`+
			`<text x="%d" y="59" text-anchor="end" class="value">%s</text>`,
		left, metric.format(minimum), left+plotWidth, metric.format(maximum),
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
				&body, `<circle class="run-dot" cx="%.1f" cy="%d" `+
					`r="1.75" fill="%s" opacity=".65"/>`,
				position(metric.run(run)), y-4+jitter, color,
			)
		}
		stats := metric.stats(benchmark)
		mean := position(stats.Mean)
		if stats.CI95Valid {
			low, high := position(stats.CI95Low), position(stats.CI95High)
			fmt.Fprintf(
				&body,
				`<path d="M %.1f %d H %.1f M %.1f %d v 10 M %.1f %d v 10" `+
					`stroke="#3b4252" stroke-width="5"/>`+
					`<path class="mean-ci" d="M %.1f %d H %.1f M %.1f %d v 10 M %.1f %d v 10" `+
					`stroke="#eceff4" stroke-width="1.5"/>`,
				low, y-4, high, low, y-9, high, y-9,
				low, y-4, high, low, y-9, high, y-9,
			)
		}
		fmt.Fprintf(
			&body,
			`<path class="mean-diamond" d="M %.1f %d l 6 6 -6 6 -6 -6 z" `+
				`fill="%s" stroke="#3b4252" stroke-width="2"/>`+
				`<text x="620" y="%d" class="value">%s</text>`,
			mean, y-10, color, y, metric.format(stats.Mean),
		)
	}
	return svgChart{
		kind: "distribution", title: metric.name, slug: chartSlug(metric.name),
		description: distributionChartDescription(metric.name),
		body:        body.String(), height: height,
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
		description: unavailableChartDescription(title, operatingSystem),
		body:        body, height: 66,
	}
}

func distributionChartDescription(metric string) string {
	return "Shows individual run values for " + metric + ". Dots are raw runs, " +
		"diamonds are arithmetic means, and whiskers are pointwise 95% confidence " +
		"intervals for the mean when n >= 2. Math: CI = mean +/- t(0.975, n-1) * " +
		"sample_sd / sqrt(n). Lower values are usually better for cost metrics."
}

func unavailableChartDescription(title, operatingSystem string) string {
	return title + " is unavailable because this metric is not collected on " + operatingSystem + "."
}
