package report

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

const htmlStart = `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Snailrace report</title><style>
:root{color-scheme:dark;--bg:#2e3440;--panel:#3b4252;--line:#4c566a;
--ink:#eceff4;--muted:#d8dee9;--accent:#88c0d0}*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--ink);font:14px/1.42 monospace}
body{font-family:"JetBrains Mono","Fira Code",SFMono-Regular,Consolas,monospace}
main{width:min(1500px,calc(100% - 24px));margin:30px auto 56px}
header{border-left:4px solid var(--accent);padding-left:16px;margin-bottom:22px}
h1,h2{letter-spacing:-.04em}h1{font-size:clamp(2rem,5vw,3.4rem);margin:0}
h2{margin:24px 0 10px}
.muted,.notes{color:var(--muted)}
.system-strip{display:flex;flex-wrap:wrap;gap:6px 20px;padding:9px 11px;
background:var(--panel);border:1px solid var(--line);border-radius:5px}
.system-strip b{color:var(--accent)}
.command-list{display:flex;flex-direction:column;border:1px solid var(--line);border-radius:5px}
.command-item{display:grid;grid-template-columns:minmax(130px,190px) 1fr;
gap:10px;padding:6px 9px;border-bottom:1px solid var(--line)}
.command-item:last-child{border-bottom:0}.command-item code{overflow-wrap:anywhere;word-break:break-word}
.winner-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:8px}
.winner{padding:10px 11px;background:#434c5e;border-top:2px solid var(--accent);border-radius:5px}
.winner strong{display:block;color:var(--accent);font-size:1.1rem;overflow-wrap:anywhere}
.rank-one{color:#a3be8c;font-weight:700}.command{white-space:pre-wrap;overflow-wrap:anywhere;word-break:break-word}
.ranking td:nth-child(2){white-space:normal;overflow-wrap:anywhere;min-width:140px}
.ranking th button{all:unset;cursor:pointer;color:inherit;font-weight:700}.ranking th button:hover{color:var(--accent)}
.raw-detail{margin:12px 0;border:1px solid var(--line);border-radius:7px;padding:10px 12px}
.raw-detail summary{cursor:pointer;color:var(--ink);font-weight:700}
table{background:var(--panel);border:1px solid var(--line)}.scroll{overflow-x:auto}
.chart-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px;margin:12px 0 22px}
.chart-column{display:flex;flex-direction:column;gap:10px;min-width:0}
.chart{min-width:0}.chart-frame{position:relative;width:100%;max-width:720px}
.chart svg{display:block;width:100%;height:auto}
.chart-help{position:absolute;top:8px;right:8px;z-index:2;width:24px;height:24px;cursor:help;
border:1px solid var(--line);border-radius:50%;background:#2e3440;color:var(--accent);font-weight:700}
.chart-help:focus{outline:2px solid var(--accent);outline-offset:2px}
.chart-tooltip{display:none;position:absolute;top:36px;right:8px;z-index:3;
max-width:min(360px,calc(100% - 16px));padding:8px 10px;border:1px solid var(--line);
border-radius:5px;background:#2e3440;color:var(--ink);box-shadow:0 8px 24px #0006}
.chart-help:hover+.chart-tooltip,.chart-help:focus+.chart-tooltip,.chart-tooltip:hover{display:block}
.badge{color:#2e3440;background:var(--accent);padding:3px 7px;border-radius:4px;font-size:.7em}
.good{color:#a3be8c}.bad{color:#bf616a}.uncertain{color:#ebcb8b}.neutral{color:#81a1c1}
.delta{display:block;font-size:.85em}.comparison td:nth-child(n+2){vertical-align:top}
table{border-collapse:collapse;width:100%}
th,td{padding:6px 9px;text-align:right;border-bottom:1px solid var(--line);white-space:nowrap}
th:first-child,td:first-child{text-align:left}code{color:var(--accent);white-space:normal}
@media(max-width:760px){main{margin-top:20px}.chart-grid{grid-template-columns:1fr}
.command-item{grid-template-columns:1fr}}
</style></head><body><main>`

func writeHTML(writer io.Writer, report model.Report) error {
	fmt.Fprint(writer, htmlStart)
	fmt.Fprintf(
		writer,
		"<header><h1>🐌 Snailrace report</h1><div class=\"muted\">%s · %s/%s</div></header>",
		report.MeasuredAt.Format("2006-01-02 15:04:05 MST"),
		html.EscapeString(report.Host.OS), html.EscapeString(report.Host.Architecture),
	)
	writeCards(writer, report)
	writeHTMLCommandLegend(writer, report)
	if len(report.Benchmarks) > 1 {
		writeHTMLRanking(writer, report)
	}
	charts := reportCharts(report)
	for _, section := range chartSections(charts) {
		writeHTMLChartSection(writer, section.title, section.charts)
	}
	if len(report.Benchmarks) > 1 {
		writeHTMLComparison(writer, report)
	}
	fmt.Fprint(writer, "<h2>Statistical detail</h2>")
	for index, benchmark := range report.Benchmarks {
		writeHTMLBenchmark(
			writer, report, report.Host.OS, benchmark, index,
			len(report.Benchmarks) > 3,
		)
	}
	fmt.Fprint(writer, `<div class="notes">`)
	for _, note := range report.Notes {
		fmt.Fprintf(writer, "<p>Note: %s</p>", html.EscapeString(note))
	}
	_, err := fmt.Fprint(writer, "</div></main></body></html>")
	return err
}

type chartSection struct {
	title  string
	charts []svgChart
}

func chartSections(charts []svgChart) []chartSection {
	definitions := []struct{ kind, title string }{
		{"ranking", "Ranking charts"},
		{"baseline", "From baseline"},
		{"tradeoff", "Tradeoffs"},
		{"distribution", "Distributions"},
		{"trend", "Measurement trends"},
	}
	var sections []chartSection
	for _, definition := range definitions {
		if definition.kind == "trend" {
			sections = append(sections, trendChartSections(definition.title, charts)...)
			continue
		}
		section := chartSection{title: definition.title}
		for _, chart := range charts {
			if chart.kind == definition.kind {
				section.charts = append(section.charts, chart)
			}
		}
		if len(section.charts) > 0 {
			sections = append(sections, section)
		}
	}
	return sections
}

func trendChartSections(title string, charts []svgChart) []chartSection {
	var sections []chartSection
	indexes := make(map[string]int)
	for _, chart := range charts {
		if chart.kind != "trend" {
			continue
		}
		sectionTitle := title
		if chart.subsection != "" {
			sectionTitle += " · " + chart.subsection
		}
		index, ok := indexes[sectionTitle]
		if !ok {
			indexes[sectionTitle] = len(sections)
			sections = append(sections, chartSection{title: sectionTitle})
			index = len(sections) - 1
		}
		sections[index].charts = append(sections[index].charts, chart)
	}
	return sections
}

func writeHTMLChartSection(writer io.Writer, title string, charts []svgChart) {
	left, right := balanceCharts(charts)
	fmt.Fprintf(writer, `<section><h2>%s</h2><div class="chart-grid">`, html.EscapeString(title))
	writeHTMLChartColumn(writer, left)
	writeHTMLChartColumn(writer, right)
	fmt.Fprint(writer, "</div></section>")
}

func writeHTMLChartColumn(writer io.Writer, charts []svgChart) {
	fmt.Fprint(writer, `<div class="chart-column">`)
	for _, chart := range charts {
		explanation := html.EscapeString(chart.explanation())
		label := html.EscapeString("Explain " + chart.accessibleTitle())
		fmt.Fprintf(
			writer,
			`<div class="chart"><div class="chart-frame">`+
				`<button type="button" class="chart-help" `+
				`aria-label="%s" title="%s">?</button>`+
				`<div class="chart-tooltip" role="tooltip">%s</div>%s</div></div>`,
			label, explanation, explanation, chart.html(),
		)
	}
	fmt.Fprint(writer, "</div>")
}

func writeCards(writer io.Writer, report model.Report) {
	fmt.Fprintf(
		writer,
		`<div class="system-strip"><span><b>%d</b> runs · %d warmups · %s</span>`+
			`<span><b>%d</b> logical CPUs · %s</span>`+
			`<span><b>%s</b> RAM · %s available</span>`+
			`<span>load <b>%s</b> · %d processes</span></div>`,
		report.Config.Runs, report.Config.Warmups,
		html.EscapeString(report.Config.Mode), report.Host.LogicalCPUs,
		html.EscapeString(report.Host.CPU),
		formatBytes(float64(report.Host.MemoryTotalBytes)),
		formatBytes(float64(report.Host.MemoryBeforeBytes)),
		html.EscapeString(report.Host.LoadBefore), report.Host.ProcessesBefore,
	)
}

func writeHTMLBenchmark(
	writer io.Writer,
	report model.Report,
	operatingSystem string,
	benchmark model.Benchmark,
	index int,
	collapsed bool,
) {
	label := htmlToolLabel(report, index, true)
	fmt.Fprint(writer, "<section>")
	if collapsed {
		fmt.Fprintf(
			writer, `<details class="raw-detail"><summary>%s</summary>`, label,
		)
	} else {
		fmt.Fprintf(
			writer, "<h2>%s</h2>", label,
		)
	}
	fmt.Fprintf(
		writer, `<pre class="command"><code>%s</code></pre>`,
		html.EscapeString(strings.Join(benchmark.Tool.Command, " ")),
	)
	fmt.Fprintf(
		writer,
		`<p class="muted">%s executable + %s linked = %s footprint `+
			`(%d files; %d dyld-cache dependencies excluded)</p>`,
		formatBytes(float64(benchmark.Tool.SizeBytes)),
		formatBytes(float64(benchmark.Tool.LinkedSizeBytes)),
		formatBytes(float64(benchmark.Tool.DiskFootprintBytes)),
		len(benchmark.Tool.LinkedFiles),
		len(benchmark.Tool.SharedCacheFiles),
	)
	fmt.Fprintf(
		writer,
		`<p class="muted">Sampling observations: %s valid, %s mean observed coverage.</p>`,
		formatCount(benchmark.Summary.ValidSampleCount.Mean),
		formatDuration(benchmark.Summary.SampleCoverageSeconds.Mean),
	)
	if !benchmarkSamplesReliable(benchmark, report.Config.IntervalMS/1000) {
		fmt.Fprint(
			writer,
			`<p class="uncertain">Sampling quality: LIMITED `+
				`(fewer than two valid samples or intervals).</p>`,
		)
	}
	fmt.Fprint(writer, `<div class="scroll"><table><thead><tr><th>Metric</th>`+
		`<th>Mean ± σ</th><th>95% CI mean</th><th>Median</th>`+
		`<th>P95</th><th>Range</th></tr></thead><tbody>`)
	for _, row := range metricRows {
		writeHTMLRow(writer, operatingSystem, row, benchmark.Summary)
	}
	fmt.Fprint(writer, "</tbody></table></div>")
	writeHTMLLinkedFiles(writer, benchmark.Tool)
	if collapsed {
		fmt.Fprint(writer, "</details>")
	}
	fmt.Fprint(writer, "</section>")
}

func writeHTMLRow(
	writer io.Writer,
	operatingSystem string,
	row metricRow,
	summary model.Summary,
) {
	if !available(row, operatingSystem) {
		fmt.Fprintf(writer, "<tr><td>%s</td><td colspan=\"5\">N/A</td></tr>", row.name)
		return
	}
	value := row.stats(summary)
	fmt.Fprintf(
		writer,
		"<tr><td>%s</td><td>%s ± %s</td><td>%s</td>"+
			"<td>%s</td><td>%s</td><td>%s .. %s</td></tr>",
		row.name, row.format(value.Mean), row.format(value.StdDev),
		formatConfidence(value, row.format),
		row.format(value.Median), row.format(value.P95),
		row.format(value.Min), row.format(value.Max),
	)
}
