package report

import (
	"fmt"
	"html"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

func writeHTMLComparison(writer io.Writer, renderer *Renderer) {
	report := renderer.report
	baselinePosition := baselineIndex(report)
	baseline := report.Benchmarks[baselinePosition]
	fmt.Fprintf(
		writer,
		`<section><h2>Detailed baseline deltas</h2><p class="muted">Baseline: `+
			`<strong>%s</strong>. Better/worse requires a pointwise paired 95%% interval; `+
			`higher/lower is descriptive.</p>`,
		htmlToolLabel(report, baselinePosition, true),
	)
	for index, candidate := range report.Benchmarks {
		if index == baselinePosition {
			continue
		}
		collapsed := len(report.Benchmarks) > 3
		if collapsed {
			fmt.Fprintf(
				writer, `<details class="raw-detail"><summary>%s vs baseline</summary>`,
				htmlToolLabel(report, index, false),
			)
		} else {
			fmt.Fprintf(writer, "<h3>%s</h3>", htmlToolLabel(report, index, false))
		}
		fmt.Fprint(writer, `<div class="scroll"><table class="comparison">`+
			`<thead><tr><th>Metric</th><th>Baseline mean</th>`+
			`<th>Candidate mean</th><th>Δ mean</th></tr></thead><tbody>`)
		writeHTMLStaticFootprint(writer, baseline, candidate)
		for metric, row := range metricRows {
			writeHTMLComparisonRow(
				writer, report.Host.OS, baseline, candidate, row,
				renderer.comparison(index, metric),
			)
		}
		fmt.Fprint(writer, "</tbody></table></div>")
		if collapsed {
			fmt.Fprint(writer, "</details>")
		}
	}
	fmt.Fprint(writer, "</section>")
}

func writeHTMLStaticFootprint(
	writer io.Writer,
	baseline, candidate model.Benchmark,
) {
	delta, class := compareStaticCost(
		float64(baseline.Tool.DiskFootprintBytes),
		float64(candidate.Tool.DiskFootprintBytes),
	)
	fmt.Fprintf(
		writer,
		`<tr><td>Linked footprint</td><td class="neutral">%s</td>`+
			`<td class="%s">%s</td>`+
			`<td><strong class="%s">%s</strong>`+
			`<span class="delta muted">static discovery</span></td></tr>`,
		formatBytes(float64(baseline.Tool.DiskFootprintBytes)),
		class, formatBytes(float64(candidate.Tool.DiskFootprintBytes)),
		class, html.EscapeString(delta),
	)
}

func writeHTMLComparisonRow(
	writer io.Writer,
	operatingSystem string,
	baseline, candidate model.Benchmark,
	row metricRow,
	delta deltaResult,
) {
	if !availableFor(row, operatingSystem, baseline.Summary, candidate.Summary) {
		fmt.Fprintf(writer, "<tr><td>%s</td><td colspan=\"3\">N/A</td></tr>", row.name)
		return
	}
	fmt.Fprintf(
		writer,
		`<tr><td>%s</td><td class="neutral">%s</td><td class="%s">%s</td>`+
			`<td><strong class="%s">%s</strong>`+
			`<span class="delta muted">%s</span></td></tr>`,
		row.name, row.format(delta.baselineMean), delta.class,
		row.format(delta.candidateMean), delta.class,
		html.EscapeString(formatDelta(delta, row)),
		html.EscapeString(formatDeltaInterval(delta, row)),
	)
}
