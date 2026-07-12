package report

import (
	"fmt"
	"html"
	"math"
	"strings"

	"perftool/internal/model"
	"perftool/internal/style"
)

func measurementTrendChart(report model.Report) svgChart {
	if len(report.Benchmarks) != 1 || len(report.Benchmarks[0].Runs) < 2 {
		return svgChart{}
	}
	metrics := chartMetricDefinitions()
	const left, right, rowHeight = 170, 605, 40
	height := 62 + len(metrics)*rowHeight
	runs := report.Benchmarks[0].Runs
	x := func(index int) float64 {
		return left + float64(index)/float64(len(runs)-1)*(right-left)
	}
	var body strings.Builder
	body.WriteString(svgChartStyle)
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="23" class="title">MEASUREMENT TRENDS · %s</text>`+
			`<text x="16" y="41" class="subtitle">independent lane scale per metric · x-axis = run iteration</text>`,
		html.EscapeString(reportToolLabel(report, 0)),
	)
	for metricIndex, metric := range metrics {
		y := 65 + metricIndex*rowHeight
		values := make([]float64, len(runs))
		minimum, maximum := math.Inf(1), math.Inf(-1)
		for index, run := range runs {
			values[index] = metric.run(run)
			minimum = math.Min(minimum, values[index])
			maximum = math.Max(maximum, values[index])
		}
		positionY := func(value float64) float64 {
			if minimum == maximum {
				return float64(y)
			}
			return float64(y+10) - (value-minimum)/(maximum-minimum)*20
		}
		color := style.Tool(metricIndex).Hex
		fmt.Fprintf(
			&body,
			`<text x="16" y="%d" class="label">%s</text>`+
				`<path d="M %d %d H %d" class="axis"/>`,
			y+4, html.EscapeString(metric.name), left, y, right,
		)
		var path strings.Builder
		for index, value := range values {
			command := "L"
			if index == 0 {
				command = "M"
			}
			fmt.Fprintf(&path, "%s %.1f %.1f ", command, x(index), positionY(value))
		}
		fmt.Fprintf(&body, `<path d="%s" fill="none" stroke="%s" stroke-width="2"/>`, path.String(), color)
		for index, value := range values {
			fmt.Fprintf(
				&body, `<circle cx="%.1f" cy="%.1f" r="3" fill="%s"/>`,
				x(index), positionY(value), color,
			)
		}
		fmt.Fprintf(
			&body, `<text x="620" y="%d" class="value">%s</text>`,
			y+4, metric.format(values[len(values)-1]),
		)
	}
	return svgChart{
		kind: "trend", title: "Measurement trends", slug: "measurement-trends",
		description: measurementTrendChartDescription(),
		body:        body.String(), height: height,
	}
}

func measurementTrendChartDescription() string {
	return "Shows run-to-run movement for each metric for a single tool. Each lane uses " +
		"its own y-scale. Math: lane y = linear interpolation between that metric's " +
		"observed min and max for this tool, so compare shape and stability within a lane, " +
		"not vertical magnitude across different metrics."
}
