package report

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"golang.org/x/term"

	"github.com/shellcell/snailrace/internal/model"
)

func writeText(writer io.Writer, renderer *Renderer) error {
	if !renderer.displayReport.Verbose {
		return writeCompactText(writer, renderer)
	}
	return writeVerboseText(writer, renderer)
}

func writeVerboseText(writer io.Writer, renderer *Renderer) error {
	report := renderer.displayReport
	var output bytes.Buffer
	w := tabwriter.NewWriter(&output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "Snailrace report\t%s\n", report.MeasuredAt.Format(
		"2006-01-02 15:04:05 MST",
	))
	fmt.Fprintf(
		w, "Host\t%s/%s, %s, %d CPUs, %s RAM\n",
		report.Host.OS, report.Host.Architecture, report.Host.CPU,
		report.Host.LogicalCPUs, formatBytes(float64(report.Host.MemoryTotalBytes)),
	)
	fmt.Fprintf(
		w, "Load before\t%s (%d processes)\n",
		report.Host.LoadBefore, report.Host.ProcessesBefore,
	)
	fmt.Fprintf(
		w, "Configuration\t%s mode, %d runs, %d warmups, %s ms sampling\n",
		report.Config.Mode, report.Config.Runs, report.Config.Warmups,
		formatNumber(report.Config.IntervalMS),
	)
	fmt.Fprintf(w, "Included dimensions\t%s\n", renderer.dimensionText)
	for _, caveat := range renderer.caveats {
		fmt.Fprintf(w, "Reliability\t%s\n", caveat)
	}
	if report.Config.IsTUI() {
		duration := "until exit"
		if report.Config.DurationSeconds > 0 {
			duration = formatDuration(report.Config.DurationSeconds)
		}
		geometry := fmt.Sprintf(
			"%dx%d fixed", report.Config.TerminalWidth,
			report.Config.TerminalHeight,
		)
		if report.Config.TerminalInherited {
			geometry = fmt.Sprintf(
				"%dx%d inherited at start",
				report.Config.TerminalWidth, report.Config.TerminalHeight,
			)
		}
		fmt.Fprintf(w, "TUI session\t%s, %s\n", duration, geometry)
	}
	fmt.Fprintln(w)
	writeTextCommandLegend(w, report)
	writeTextFailures(w, report)
	if len(report.Benchmarks) > 1 {
		writeTextRankingWith(w, report, renderer.ranking)
	}
	if len(report.Benchmarks) > 1 {
		fmt.Fprintln(w, "Detailed baseline deltas\t")
		writeTextComparison(w, renderer)
	}
	fmt.Fprintln(w, "Statistical detail\t")
	for index, benchmark := range report.Benchmarks {
		writeTextBenchmark(
			w, report.Host.OS, report.Config.IntervalMS/1000,
			benchmark, index == baselineIndex(report),
		)
	}
	for _, note := range report.Notes {
		fmt.Fprintf(w, "Note\t%s\n", note)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	text := output.String()
	if writerIsTerminal(writer) {
		text = colorizeText(text, report)
	}
	_, err := io.WriteString(writer, text)
	return err
}

func writeTextBenchmark(
	w io.Writer,
	operatingSystem string,
	intervalSeconds float64,
	benchmark model.Benchmark,
	baseline bool,
) {
	label := benchmark.Tool.Name
	if baseline {
		label += " [BASELINE]"
	}
	fmt.Fprintf(w, "%s\t\n", label)
	commandLines := wrapText(fullCommand(benchmark), 100)
	for index, line := range commandLines {
		field := ""
		if index == 0 {
			field = "Command"
		}
		fmt.Fprintf(w, "%s\t%s\n", field, line)
	}
	fmt.Fprintf(
		w, "Disk footprint\t%s executable + %s linked = %s (%d files; %d dyld-cache dependencies excluded)\n",
		formatBytes(float64(benchmark.Tool.SizeBytes)),
		formatBytes(float64(benchmark.Tool.LinkedSizeBytes)),
		formatBytes(float64(benchmark.Tool.DiskFootprintBytes)),
		len(benchmark.Tool.LinkedFiles),
		len(benchmark.Tool.SharedCacheFiles),
	)
	fmt.Fprintf(
		w, "Sampling observations\t%s valid, %s mean observed coverage\n",
		formatCount(benchmark.Summary.ValidSampleCount.Mean),
		formatDuration(benchmark.Summary.SampleCoverageSeconds.Mean),
	)
	if !model.SamplingReliable(benchmark, intervalSeconds) {
		fmt.Fprintln(w, "Sampling quality\tLIMITED: fewer than two valid samples or intervals")
	}
	fmt.Fprintln(w, "Metric\tMean ± σ\t95% CI mean\tMedian\tP95\tRange")
	for _, row := range metricCatalog {
		if !availableFor(row, operatingSystem, benchmark.Summary) {
			fmt.Fprintf(w, "%s\tN/A\tN/A\tN/A\tN/A\tN/A\n", row.name)
			continue
		}
		value := row.stats(benchmark.Summary)
		fmt.Fprintf(
			w, "%s\t%s ± %s\t%s\t%s\t%s\t%s .. %s\n",
			row.name, row.format(value.Mean), row.format(value.StdDev),
			formatConfidence(value, row.format),
			row.format(value.Median), row.format(value.P95),
			row.format(value.Min), row.format(value.Max),
		)
	}
	fmt.Fprintln(w)
}

func writerIsTerminal(writer io.Writer) bool {
	file, ok := writer.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(file.Fd()))
}
