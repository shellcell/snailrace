package report

import (
	"fmt"
	"io"
	"strings"

	"perftool/internal/model"
)

func writeMarkdown(writer io.Writer, report model.Report) error {
	return WriteMarkdownWithCharts(writer, report, nil, "")
}

func WriteMarkdownWithCharts(
	writer io.Writer,
	report model.Report,
	charts []ChartArtifact,
	chartDirectory string,
) error {
	fmt.Fprintf(
		writer,
		"# 🐌 Snailrace Report\n\nMeasured %s on `%s/%s`. "+
			"%d measured runs after %d warmups in `%s` mode.\n\n",
		report.MeasuredAt.Format("2006-01-02 15:04:05 MST"),
		report.Host.OS, report.Host.Architecture,
		report.Config.Runs, report.Config.Warmups, report.Config.Mode,
	)
	writeMarkdownCommandLegend(writer, report)
	if len(report.Benchmarks) > 1 {
		writeMarkdownRanking(writer, report)
	}
	writeMarkdownCharts(writer, charts, chartDirectory)
	if len(report.Benchmarks) > 1 {
		writeMarkdownComparison(writer, report)
	}
	fmt.Fprintln(writer, "## Statistical Detail")
	for index, benchmark := range report.Benchmarks {
		writeMarkdownBenchmark(
			writer, report.Host.OS, benchmark, index == baselineIndex(report),
		)
	}
	fmt.Fprintln(writer, "## Environment")
	fmt.Fprintf(
		writer,
		"\n- CPU: %s (%d logical CPUs)\n"+
			"- Memory: %s total, %s available before, %s available after\n"+
			"- Load average before: %s\n- Processes before: %d\n\n",
		escapeMarkdown(report.Host.CPU), report.Host.LogicalCPUs,
		formatBytes(float64(report.Host.MemoryTotalBytes)),
		formatBytes(float64(report.Host.MemoryBeforeBytes)),
		formatBytes(float64(report.Host.MemoryAfterBytes)),
		report.Host.LoadBefore, report.Host.ProcessesBefore,
	)
	for _, note := range report.Notes {
		fmt.Fprintf(writer, "> %s\n\n", escapeMarkdown(note))
	}
	return nil
}

func writeMarkdownBenchmark(
	writer io.Writer,
	operatingSystem string,
	benchmark model.Benchmark,
	baseline bool,
) {
	label := benchmark.Tool.Name
	if baseline {
		label += " [BASELINE]"
	}
	fmt.Fprintf(
		writer,
		"### %s\n\nCommand:\n\n```sh\n%s\n```\n\nExecutable SHA-256: `%s`  \n"+
			"Disk footprint: %s executable + %s linked = %s (%d libraries)\n\n",
		escapeMarkdown(label),
		strings.Join(benchmark.Tool.Command, " "),
		benchmark.Tool.SHA256,
		formatBytes(float64(benchmark.Tool.SizeBytes)),
		formatBytes(float64(benchmark.Tool.LinkedSizeBytes)),
		formatBytes(float64(benchmark.Tool.DiskFootprintBytes)),
		len(benchmark.Tool.LinkedFiles),
	)
	fmt.Fprintln(
		writer,
		"| Metric | Mean ± σ | 95% CI mean | Median | P95 | Range |",
	)
	fmt.Fprintln(writer, "|---|---:|---:|---:|---:|---:|")
	for _, row := range metricRows {
		if !available(row, operatingSystem) {
			fmt.Fprintf(writer, "| %s | N/A | N/A | N/A | N/A | N/A |\n", row.name)
			continue
		}
		value := row.stats(benchmark.Summary)
		fmt.Fprintf(
			writer, "| %s | %s ± %s | [%s, %s] | %s | %s | %s .. %s |\n",
			row.name, row.format(value.Mean), row.format(value.StdDev),
			row.format(value.CI95Low), row.format(value.CI95High),
			row.format(value.Median), row.format(value.P95),
			row.format(value.Min), row.format(value.Max),
		)
	}
	fmt.Fprintln(writer)
}

func escapeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.ReplaceAll(value, "`", "\\`")
}
