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
	toolColorHex := toolColor(benchmarkIndex)
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
			minimum, maximum := trendMetricRange(runs, metric)
			color := style.Tool(int(metric.id)).Hex
			var path strings.Builder
			path.Grow(len(runs) * 20)
			started := false
			for index, run := range runs {
				if !chartRunAvailable(metric, run) {
					started = false
					continue
				}
				value := metric.run(run)
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
			for index, run := range runs {
				if !chartRunAvailable(metric, run) {
					continue
				}
				value := metric.run(run)
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
				legendY, color, html.EscapeString(clip(metric.chartName, 24)), legendY+12,
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

func trendMetricRange(runs []model.Run, metric chartMetric) (float64, float64) {
	minimum, maximum := math.Inf(1), math.Inf(-1)
	for _, run := range runs {
		if !chartRunAvailable(metric, run) {
			continue
		}
		value := metric.run(run)
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
	byRow := make(map[metricID]chartMetric, len(metrics))
	for _, metric := range metrics {
		byRow[metric.id] = metric
	}
	switch group.name {
	case "PERFORMANCE":
		return trendLaneDefinitions(byRow, trendLaneSpec{"Wall time", []metricID{metricWall}})
	case "CPU COST":
		return trendLaneDefinitions(
			byRow,
			trendLaneSpec{"CPU time", []metricID{metricCPUTotal, metricCPUUser, metricCPUSystem}},
			trendLaneSpec{"Average CPU", []metricID{metricAverageCPU}},
		)
	case "MEMORY COST":
		return trendLaneDefinitions(
			byRow,
			trendLaneSpec{"Resident memory", []metricID{
				metricOSMaxRSS, metricPeakResident, metricMeanResident,
			}},
			trendLaneSpec{"Physical footprint", []metricID{metricPhysicalFootprint}},
			trendLaneSpec{"Virtual memory", []metricID{metricPeakVirtual}},
		)
	case "PROCESS STRUCTURE":
		return trendLaneDefinitions(
			byRow,
			trendLaneSpec{"Processes and threads", []metricID{
				metricPeakThreads, metricPeakProcesses,
			}},
			trendLaneSpec{"FD references", []metricID{metricPeakFDs}},
		)
	default:
		lanes := make([]trendLane, 0, len(metrics))
		for _, metric := range metrics {
			lanes = append(lanes, trendLane{
				name: metric.chartName, metrics: []chartMetric{metric}, format: metric.format,
			})
		}
		return lanes
	}
}

type trendLaneSpec struct {
	name string
	rows []metricID
}

func trendLaneDefinitions(byRow map[metricID]chartMetric, specs ...trendLaneSpec) []trendLane {
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
		stats := metric.stats(report.Benchmarks[benchmarkIndex].Summary)
		if stats.N > 0 && finiteNumber(stats.Mean) && availableFor(
			metric, report.Host.OS,
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
		names = append(names, metric.chartName)
	}
	return strings.Join(names, ", ")
}
