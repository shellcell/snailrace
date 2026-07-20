package report

import (
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

type tradeoffMetric struct {
	name   string
	format func(float64) string
	value  func(model.Benchmark) float64
}

func tradeoffCharts(report model.Report, ranking rankingData) []svgChart {
	if len(report.Benchmarks) < 2 {
		return nil
	}
	primary := tradeoffMetric{"Time", formatDuration, func(b model.Benchmark) float64 {
		return b.Summary.WallSeconds.Mean
	}}
	if report.Config.FixedDurationTUI() {
		primary = tradeoffMetric{"Average CPU", formatPercent, func(b model.Benchmark) float64 {
			return b.Summary.AverageCPUPercent.Mean
		}}
	}
	cpu := tradeoffMetric{"Total CPU", formatDuration, func(b model.Benchmark) float64 {
		return b.Summary.CPUTotalSeconds.Mean
	}}
	ram := tradeoffMetric{"RAM aggregate", formatBytes, func(b model.Benchmark) float64 {
		mean := b.Summary.MeanResidentBytes.Mean
		peak := b.Summary.PeakResidentBytes.Mean
		if mean <= 0 || peak <= 0 {
			return 0
		}
		return math.Sqrt(mean) * math.Sqrt(peak)
	}}
	linked := tradeoffMetric{"Linked size", formatBytes, func(b model.Benchmark) float64 {
		return float64(b.Tool.DiskFootprintBytes)
	}}
	pairs := [][2]tradeoffMetric{{primary, cpu}, {primary, linked}}
	if ranking.RAMAvailable {
		pairs = append(pairs, [2]tradeoffMetric{primary, ram}, [2]tradeoffMetric{cpu, ram})
	}
	charts := make([]svgChart, 0, len(pairs))
	for _, pair := range pairs {
		charts = append(charts, tradeoffChart(report, pair[0], pair[1]))
	}
	return charts
}

func tradeoffChart(report model.Report, xMetric, yMetric tradeoffMetric) svgChart {
	const left, top, plotWidth, plotHeight = 62, 62, 390, 240
	xMaximum, yMaximum := 0.0, 0.0
	for _, benchmark := range report.Benchmarks {
		xValue, yValue := xMetric.value(benchmark), yMetric.value(benchmark)
		if !finiteNumber(xValue) || !finiteNumber(yValue) ||
			xValue < 0 || yValue < 0 {
			return svgChart{}
		}
		xMaximum = math.Max(xMaximum, xValue)
		yMaximum = math.Max(yMaximum, yValue)
	}
	if xMaximum <= 0 {
		xMaximum = 1
	}
	if yMaximum <= 0 {
		yMaximum = 1
	}
	height := max(340, 78+len(report.Benchmarks)*26)
	var body strings.Builder
	body.WriteString(svgChartStyle)
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="23" class="title">TRADEOFF · %s × %s</text>`+
			`<text x="16" y="41" class="subtitle">lower-left is better · x: %s · y: %s</text>`+
			`<path d="M %d %d V %d H %d" fill="none" class="axis"/>`+
			`<text x="%d" y="322" class="value">0</text>`+
			`<text x="%d" y="322" text-anchor="end" class="value">%s</text>`+
			`<text x="%d" y="%d" class="value">%s</text>`,
		html.EscapeString(xMetric.name), html.EscapeString(yMetric.name),
		html.EscapeString(xMetric.name), html.EscapeString(yMetric.name),
		left, top, top+plotHeight, left+plotWidth, left, left+plotWidth,
		xMetric.format(xMaximum), left, top-5, yMetric.format(yMaximum),
	)
	for index, benchmark := range report.Benchmarks {
		xValue, yValue := xMetric.value(benchmark), yMetric.value(benchmark)
		x := left + xValue/xMaximum*plotWidth
		y := top + plotHeight - yValue/yMaximum*plotHeight
		color := toolColor(index)
		legendY := 67 + index*26
		fmt.Fprintf(
			&body,
			`<circle cx="%.1f" cy="%.1f" r="6" fill="%s" stroke="#2e3440" stroke-width="2"/>`+
				`<circle cx="480" cy="%d" r="5" fill="%s"/>`+
				`<text x="492" y="%d" class="label" style="fill:%s">%s</text>`+
				`<text x="704" y="%d" text-anchor="end" class="value">%s · %s</text>`,
			x, y, color, legendY-4, color, legendY, color,
			html.EscapeString(clip(reportToolLabel(report, index), 13)), legendY,
			xMetric.format(xValue), yMetric.format(yValue),
		)
	}
	title := xMetric.name + " vs " + yMetric.name
	return svgChart{
		kind: "tradeoff", title: title, slug: chartSlug(title),
		description: tradeoffChartDescription(xMetric.name, yMetric.name),
		body:        body.String(), height: height,
	}
}

func tradeoffChartDescription(xMetric, yMetric string) string {
	return "Plots " + xMetric + " against " + yMetric + " for each tool. " +
		"X: " + tradeoffMetricFormula(xMetric) + " Y: " + tradeoffMetricFormula(yMetric) + " " +
		"Points closer to the lower-left use less of both resources. " +
		"This chart uses point estimates only and does not prove dominance when " +
		"measurements overlap."
}

func tradeoffMetricFormula(metric string) string {
	switch metric {
	case "Time":
		return "mean wall time."
	case "Average CPU":
		return "mean average CPU percent."
	case "Total CPU":
		return "mean(user CPU + system CPU)."
	case "RAM aggregate":
		return "sqrt(mean of per-run mean RSS * mean of per-run peak RSS)."
	case "Linked size":
		return "executable bytes + linked-library bytes."
	}
	return "point estimate."
}
