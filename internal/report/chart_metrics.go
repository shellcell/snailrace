package report

import "github.com/shellcell/snailrace/internal/model"

func buildReportCharts(renderer *Renderer) []svgChart {
	report := renderer.displayReport
	if len(report.Benchmarks) == 0 {
		return nil
	}
	groups := renderer.chartGroups
	var charts []svgChart
	if len(report.Benchmarks) > 1 {
		charts = append(charts, rankingCharts(report, renderer.ranking)...)
		for _, group := range groups {
			for _, metric := range group.metrics {
				chart := deltaForestChart(renderer, chartGroup{
					name: metric.chartName, metrics: []chartMetric{metric},
				})
				if chart.height > 0 {
					charts = append(charts, chart)
				}
			}
		}
		charts = append(charts, tradeoffCharts(report, renderer.ranking)...)
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
	charts = append(charts, measurementTrendCharts(report, groups)...)
	return charts
}

func chartGroups(report model.Report) []chartGroup {
	groups := []chartGroup{
		{"PERFORMANCE", chartMetrics(metricWall)},
		{"CPU COST", chartMetrics(
			metricCPUTotal, metricCPUUser, metricCPUSystem, metricAverageCPU,
		)},
		{"MEMORY COST", chartMetrics(
			metricMeanResident, metricPeakResident, metricOSMaxRSS,
			metricPhysicalFootprint, metricPeakVirtual,
		)},
		{"PROCESS STRUCTURE", chartMetrics(
			metricPeakProcesses, metricPeakThreads, metricPeakFDs,
		)},
	}
	if report.Config.FixedDurationTUI() {
		groups = groups[1:]
	}
	return groups
}

func chartMetrics(ids ...metricID) []chartMetric {
	metrics := make([]chartMetric, len(ids))
	for index, id := range ids {
		metrics[index] = metricCatalog[id]
	}
	return metrics
}

func chartRunAvailable(metric chartMetric, run model.Run) bool {
	return (metric.id != metricPhysicalFootprint || run.PhysicalFootprintValid) &&
		finiteNumber(metric.run(run))
}
