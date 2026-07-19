package report

import (
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/style"
)

type trendLane struct {
	name    string
	metrics []chartMetric
	format  func(float64) string
}

func measurementTrendCharts(report model.Report, groups []chartGroup) []svgChart {
	if len(report.Benchmarks) == 0 {
		return nil
	}
	charts := make([]svgChart, 0, len(report.Benchmarks)*len(groups))
	for benchmarkIndex, benchmark := range report.Benchmarks {
		if len(benchmark.Runs) < 2 {
			continue
		}
		for _, group := range groups {
			chart := measurementTrendChart(report, benchmarkIndex, group)
			if chart.height > 0 {
				charts = append(charts, chart)
			}
		}
	}
	return charts
}

func measurementTrendChart(report model.Report, benchmarkIndex int, group chartGroup) svgChart {
	lanes := trendLanes(report, benchmarkIndex, group)
	if len(lanes) == 0 {
		return svgChart{}
	}
	const left, right, laneHeight = 92, 500, 108
	height := 84 + len(lanes)*laneHeight
	runs := report.Benchmarks[benchmarkIndex].Runs
	x := func(index int) float64 {
		return left + float64(index)/float64(len(runs)-1)*(right-left)
	}
	var body strings.Builder
	metricCount := 0
	for _, lane := range lanes {
		metricCount += len(lane.metrics)
	}
	body.Grow(1536 + len(runs)*metricCount*96)
	toolLabel := reportToolLabel(report, benchmarkIndex)
	toolColorHex := toolColor(report, benchmarkIndex)
	body.WriteString(svgChartStyle)
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="23" class="title">MEASUREMENT TRENDS · %s</text>`+
			`<text x="16" y="41" class="subtitle" style="fill:%s">%s</text>`,
		html.EscapeString(group.name), toolColorHex, html.EscapeString(clip(toolLabel, 42)),
	)
	for laneIndex, lane := range lanes {
		top := 98 + laneIndex*laneHeight
		bottom := top + 42
		middle := (top + bottom) / 2
		minimum, maximum := trendLaneRange(runs, lane.metrics)
		midValue := (minimum + maximum) / 2
		positionY := func(value float64) float64 {
			if minimum == maximum {
				return float64(top+bottom) / 2
			}
			return float64(bottom) - (value-minimum)/(maximum-minimum)*float64(bottom-top)
		}
		fmt.Fprintf(
			&body,
			`<text x="%d" y="%d" class="label">%s</text>`+
				`<rect x="%d" y="%d" width="%d" height="%d" fill="#2e3440" opacity=".25"/>`+
				`<path d="M %d %d H %d M %d %d H %d M %d %d H %d" class="grid"/>`+
				`<path d="M %d %d V %d H %d" fill="none" class="axis"/>`+
				`<text x="%d" y="%d" text-anchor="end" class="value">%s</text>`+
				`<text x="%d" y="%d" text-anchor="end" class="value">%s</text>`+
				`<text x="%d" y="%d" text-anchor="end" class="value">%s</text>`+
				`<text x="%d" y="%d" class="value">run 1</text>`+
				`<text x="%d" y="%d" text-anchor="end" class="value">run %d</text>`,
			left, top-10, html.EscapeString(lane.name), left, top, right-left, bottom-top,
			left, top, right, left, middle, right, left, bottom, right,
			left, top, bottom, right, left-8, top+4, lane.format(maximum),
			left-8, middle+4, lane.format(midValue), left-8, bottom+4, lane.format(minimum),
			left, bottom+18, right, bottom+18, len(runs),
		)
		for metricIndex, metric := range lane.metrics {
			values := trendValues(runs, metric)
			minimum, maximum := valueRange(values)
			color := style.Tool(metric.row).Hex
			var path strings.Builder
			path.Grow(len(values) * 20)
			started := false
			for index, value := range values {
				if !finiteChartValue(value) {
					started = false
					continue
				}
				command := "L"
				if !started {
					command = "M"
				}
				path.WriteString(command)
				path.WriteByte(' ')
				writeChartFloat(&path, x(index))
				path.WriteByte(' ')
				writeChartFloat(&path, positionY(value))
				path.WriteByte(' ')
				started = true
			}
			fmt.Fprintf(
				&body,
				`<path class="trend-line" d="%s" fill="none" stroke="%s" stroke-width="2"/>`,
				path.String(), color,
			)
			for index, value := range values {
				if !finiteChartValue(value) {
					continue
				}
				body.WriteString(`<circle class="trend-dot" cx="`)
				writeChartFloat(&body, x(index))
				body.WriteString(`" cy="`)
				writeChartFloat(&body, positionY(value))
				body.WriteString(`" r="1.5" fill="`)
				body.WriteString(color)
				body.WriteString(`"/>`)
			}
			legendY := top + 4 + metricIndex*26
			fmt.Fprintf(
				&body,
				`<text x="520" y="%d" class="label" style="fill:%s">%s</text>`+
					`<text x="520" y="%d" class="value">%s ... %s</text>`,
				legendY, color, html.EscapeString(clip(metric.name, 24)), legendY+12,
				metric.format(minimum), metric.format(maximum),
			)
		}
	}
	title := trendChartTitle(report, benchmarkIndex, group.name)
	return svgChart{
		kind: "trend", title: title, slug: chartSlug(title), subsection: reportToolLabel(report, benchmarkIndex),
		description: measurementTrendChartDescription(report, benchmarkIndex, group.name, lanes),
		body:        body.String(), height: height,
	}
}

func trendValues(runs []model.Run, metric chartMetric) []float64 {
	values := make([]float64, len(runs))
	for index, run := range runs {
		if chartRunAvailable(metric, run) {
			values[index] = metric.run(run)
		} else {
			values[index] = math.NaN()
		}
	}
	return values
}

func valueRange(values []float64) (float64, float64) {
	minimum, maximum := math.Inf(1), math.Inf(-1)
	for _, value := range values {
		if !finiteChartValue(value) {
			continue
		}
		minimum = math.Min(minimum, value)
		maximum = math.Max(maximum, value)
	}
	if math.IsInf(minimum, 0) || math.IsInf(maximum, 0) {
		return 0, 0
	}
	return minimum, maximum
}

func trendLaneRange(runs []model.Run, metrics []chartMetric) (float64, float64) {
	minimum, maximum := math.Inf(1), math.Inf(-1)
	for _, metric := range metrics {
		for _, run := range runs {
			if !chartRunAvailable(metric, run) {
				continue
			}
			value := metric.run(run)
			minimum = math.Min(minimum, value)
			maximum = math.Max(maximum, value)
		}
	}
	if math.IsInf(minimum, 0) || math.IsInf(maximum, 0) {
		return 0, 1
	}
	return minimum, maximum
}

func trendLanes(report model.Report, benchmarkIndex int, group chartGroup) []trendLane {
	metrics := trendMetrics(report, benchmarkIndex, group.metrics)
	byRow := make(map[int]chartMetric, len(metrics))
	for _, metric := range metrics {
		byRow[metric.row] = metric
	}
	switch group.name {
	case "PERFORMANCE":
		return trendLaneDefinitions(byRow, trendLaneSpec{"Wall time", []int{0}})
	case "CPU COST":
		return trendLaneDefinitions(
			byRow,
			trendLaneSpec{"CPU time", []int{1, 2, 3}},
			trendLaneSpec{"Average CPU", []int{4}},
		)
	case "MEMORY COST":
		return trendLaneDefinitions(
			byRow,
			trendLaneSpec{"Resident memory", []int{7, 6, 5}},
			trendLaneSpec{"Physical footprint", []int{8}},
			trendLaneSpec{"Virtual memory", []int{9}},
		)
	case "PROCESS STRUCTURE":
		return trendLaneDefinitions(
			byRow,
			trendLaneSpec{"Processes and threads", []int{11, 10}},
			trendLaneSpec{"FD references", []int{12}},
		)
	default:
		lanes := make([]trendLane, 0, len(metrics))
		for _, metric := range metrics {
			lanes = append(lanes, trendLane{
				name: metric.name, metrics: []chartMetric{metric}, format: metric.format,
			})
		}
		return lanes
	}
}

type trendLaneSpec struct {
	name string
	rows []int
}

func trendLaneDefinitions(byRow map[int]chartMetric, specs ...trendLaneSpec) []trendLane {
	lanes := make([]trendLane, 0, len(specs))
	for _, spec := range specs {
		metrics := make([]chartMetric, 0, len(spec.rows))
		for _, row := range spec.rows {
			metric, ok := byRow[row]
			if ok {
				metrics = append(metrics, metric)
			}
		}
		if len(metrics) > 0 {
			lanes = append(lanes, trendLane{name: spec.name, metrics: metrics, format: metrics[0].format})
		}
	}
	return lanes
}

func trendMetrics(
	report model.Report, benchmarkIndex int, metrics []chartMetric,
) []chartMetric {
	result := make([]chartMetric, 0, len(metrics))
	for _, metric := range metrics {
		stats := metricRows[metric.row].stats(report.Benchmarks[benchmarkIndex].Summary)
		if stats.N > 0 && finiteChartValue(stats.Mean) && availableFor(
			metricRows[metric.row], report.Host.OS,
			report.Benchmarks[benchmarkIndex].Summary,
		) {
			result = append(result, metric)
		}
	}
	return result
}

func trendChartTitle(report model.Report, benchmarkIndex int, group string) string {
	title := trendGroupTitle(group)
	if len(report.Benchmarks) > 1 {
		title += " · " + reportToolLabel(report, benchmarkIndex)
	}
	return title
}

func trendGroupTitle(group string) string {
	switch group {
	case "PERFORMANCE":
		return "Performance trends"
	case "CPU COST":
		return "CPU cost trends"
	case "MEMORY COST":
		return "Memory cost trends"
	case "PROCESS STRUCTURE":
		return "Process structure trends"
	default:
		return group + " trends"
	}
}

func measurementTrendChartDescription(
	report model.Report,
	benchmarkIndex int,
	group string,
	lanes []trendLane,
) string {
	return "Shows run-to-run movement for " + strings.ToLower(group) +
		" metrics for " + reportToolLabel(report, benchmarkIndex) +
		". Included here: " + trendLaneNames(lanes) +
		". Series in the same lane share one y-scale; different lanes use independent " +
		"scales with min/mid/max y-axis labels. Math: y = linear interpolation between " +
		"the lane's observed min and max."
}

func trendLaneNames(lanes []trendLane) string {
	parts := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		parts = append(parts, lane.name+" ("+trendMetricNames(lane.metrics)+")")
	}
	return strings.Join(parts, "; ")
}

func trendMetricNames(metrics []chartMetric) string {
	names := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		names = append(names, metric.name)
	}
	return strings.Join(names, ", ")
}
