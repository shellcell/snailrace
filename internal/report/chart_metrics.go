package report

import "perftool/internal/model"

func reportCharts(report model.Report) []svgChart {
	groups := chartGroups(report)
	charts := rankingCharts(report)
	if len(report.Benchmarks) > 1 {
		for _, group := range groups {
			for _, metric := range group.metrics {
				chart := deltaForestChart(report, chartGroup{
					name: metric.name, metrics: []chartMetric{metric},
				})
				if chart.height > 0 {
					charts = append(charts, chart)
				}
			}
		}
		charts = append(charts, tradeoffCharts(report)...)
	}
	for _, group := range groups {
		for _, metric := range group.metrics {
			chart := absoluteDistributionChart(report, metric)
			if chart.height > 0 {
				charts = append(charts, chart)
			}
		}
	}
	if chart := staticCostChart(
		report, "LINKED DISK FOOTPRINT", func(benchmark model.Benchmark) float64 {
			return float64(benchmark.Tool.DiskFootprintBytes)
		}, formatBytes,
	); chart.height > 0 {
		charts = append(charts, chart)
	}
	if chart := measurementTrendChart(report); chart.height > 0 {
		charts = append(charts, chart)
	}
	return charts
}

func chartGroups(report model.Report) []chartGroup {
	metrics := chartMetricDefinitions()
	groups := []chartGroup{
		{"PERFORMANCE", metrics[0:1]},
		{"CPU COST", metrics[1:5]},
		{"MEMORY COST", metrics[5:9]},
		{"PROCESS STRUCTURE", metrics[9:12]},
	}
	if report.Config.Mode == "tui" && report.Config.DurationSeconds > 0 {
		groups = groups[1:]
	}
	return groups
}

func chartMetricDefinitions() []chartMetric {
	return []chartMetric{
		{"Wall time", formatDuration,
			func(b model.Benchmark) model.Stats { return b.Summary.WallSeconds },
			func(r model.Run) float64 { return r.WallSeconds }, 0},
		{"Total CPU", formatDuration,
			func(b model.Benchmark) model.Stats { return b.Summary.CPUTotalSeconds },
			func(r model.Run) float64 { return r.CPUUserSeconds + r.CPUSystemSeconds }, 1},
		{"User CPU", formatDuration,
			func(b model.Benchmark) model.Stats { return b.Summary.CPUUserSeconds },
			func(r model.Run) float64 { return r.CPUUserSeconds }, 2},
		{"System CPU", formatDuration,
			func(b model.Benchmark) model.Stats { return b.Summary.CPUSystemSeconds },
			func(r model.Run) float64 { return r.CPUSystemSeconds }, 3},
		{"Average CPU", formatPercent,
			func(b model.Benchmark) model.Stats { return b.Summary.AverageCPUPercent },
			func(r model.Run) float64 { return r.AverageCPUPercent }, 4},
		{"Mean RSS", formatBytes,
			func(b model.Benchmark) model.Stats { return b.Summary.MeanResidentBytes },
			func(r model.Run) float64 { return r.MeanResidentBytes }, 5},
		{"Peak RSS", formatBytes,
			func(b model.Benchmark) model.Stats { return b.Summary.PeakResidentBytes },
			func(r model.Run) float64 { return r.PeakResidentBytes }, 6},
		{"OS-reported max RSS", formatBytes,
			func(b model.Benchmark) model.Stats { return b.Summary.OSMaxRSSBytes },
			func(r model.Run) float64 { return r.OSMaxRSSBytes }, 7},
		{"Peak virtual memory", formatBytes,
			func(b model.Benchmark) model.Stats { return b.Summary.PeakVirtualBytes },
			func(r model.Run) float64 { return r.PeakVirtualBytes }, 8},
		{"Peak processes", formatCount,
			func(b model.Benchmark) model.Stats { return b.Summary.PeakProcesses },
			func(r model.Run) float64 { return r.PeakProcesses }, 9},
		{"Peak threads", formatCount,
			func(b model.Benchmark) model.Stats { return b.Summary.PeakThreads },
			func(r model.Run) float64 { return r.PeakThreads }, 10},
		{"Peak FD references", formatCount,
			func(b model.Benchmark) model.Stats { return b.Summary.PeakFileDescriptors },
			func(r model.Run) float64 { return r.PeakFileDescriptors }, 11},
	}
}
