package report

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

const svgReportWidth = 1500

const svgReportStyle = `<style>` +
	`text{font-family:"JetBrains Mono","Fira Code",SFMono-Regular,Consolas,monospace}` +
	`.h1{fill:#eceff4;font-size:42px;font-weight:700}` +
	`.h2{fill:#eceff4;font-size:23px;font-weight:700}` +
	`.h3{fill:#e5e9f0;font-size:17px;font-weight:700}` +
	`.body{fill:#e5e9f0;font-size:13px}.muted{fill:#d8dee9;font-size:12px}` +
	`.good{fill:#a3be8c}.bad{fill:#bf616a}.uncertain{fill:#ebcb8b}` +
	`.neutral{fill:#81a1c1}.rule{stroke:#4c566a}.panel{fill:#3b4252}` +
	`</style>`

func writeSVG(writer io.Writer, report model.Report) error {
	return writeSVGRenderer(writer, NewRenderer(report))
}

func writeSVGRenderer(writer io.Writer, renderer *Renderer) error {
	report := renderer.report
	checked := newErrorWriter(writer)
	writer = checked
	var content strings.Builder
	y := svgHeader(&content, renderer)
	y = svgCommandLegend(&content, report, y)
	charts := renderer.reportCharts()
	if failures := failureChart(report); failures.height > 0 {
		charts = append([]svgChart{failures}, charts...)
	}
	y = svgCharts(&content, charts, y)
	height := y + 36
	fmt.Fprintf(
		writer,
		`<svg xmlns="http://www.w3.org/2000/svg" role="img" `+
			`aria-labelledby="title description" viewBox="0 0 %d %d">`+
			`<title id="title">Snailrace charts</title>`+
			`<desc id="description">Benchmark command legend and charts</desc>`+
			`<rect width="100%%" height="100%%" fill="#2e3440"/>%s%s</svg>`,
		svgReportWidth, height, svgReportStyle, content.String(),
	)
	return checked.Err()
}

func svgHeader(output *strings.Builder, renderer *Renderer) int {
	report := renderer.report
	caveats := renderer.caveats
	panelHeight := 96 + len(caveats)*22
	nextSection := 254 + len(caveats)*22
	if report.Config.Mode == "tui" {
		panelHeight += 24
		nextSection += 24
	}
	fmt.Fprint(output, `<rect x="24" y="28" width="6" height="72" fill="#88c0d0"/>`)
	fmt.Fprint(output, `<text id="report-title" x="48" y="65" class="h1">🐌 SNAILRACE CHARTS</text>`)
	fmt.Fprintf(
		output, `<text x="50" y="92" class="muted">%s · %s/%s · %s mode</text>`,
		html.EscapeString(report.MeasuredAt.Format("2006-01-02 15:04:05 MST")),
		html.EscapeString(report.Host.OS), html.EscapeString(report.Host.Architecture),
		html.EscapeString(report.Config.Mode),
	)
	fmt.Fprintf(
		output,
		`<rect x="24" y="122" width="1452" height="%d" rx="8" class="panel"/>`+
			`<text x="44" y="150" class="body">CPU  %s · %d logical cores</text>`+
			`<text x="44" y="177" class="muted">RAM %s · available %s before / %s after</text>`+
			`<text x="820" y="150" class="body">%d runs · %d warmups · %s ms sampling</text>`,
		panelHeight, html.EscapeString(clip(report.Host.CPU, 72)),
		report.Host.LogicalCPUs,
		formatBytes(float64(report.Host.MemoryTotalBytes)),
		formatBytes(float64(report.Host.MemoryBeforeBytes)),
		formatBytes(float64(report.Host.MemoryAfterBytes)),
		report.Config.Runs, report.Config.Warmups,
		formatNumber(report.Config.IntervalMS),
	)
	fmt.Fprintf(
		output, `<text x="820" y="177" class="muted">load %s · %d processes · kernel %s</text>`,
		html.EscapeString(report.Host.LoadBefore), report.Host.ProcessesBefore,
		html.EscapeString(clip(report.Host.Kernel, 32)),
	)
	fmt.Fprintf(
		output, `<text x="44" y="202" class="muted">balanced dimensions %s</text>`,
		html.EscapeString(renderer.dimensionText),
	)
	for index, caveat := range caveats {
		fmt.Fprintf(
			output, `<text x="44" y="%d" class="uncertain">reliability %s</text>`,
			227+index*22, html.EscapeString(caveat),
		)
	}
	if report.Config.Mode == "tui" {
		duration := "until exit"
		if report.Config.DurationSeconds > 0 {
			duration = formatDuration(report.Config.DurationSeconds)
		}
		fmt.Fprintf(
			output, `<text x="44" y="%d" class="muted">TUI %s · %dx%d</text>`,
			227+len(caveats)*22, duration,
			report.Config.TerminalWidth, report.Config.TerminalHeight,
		)
	}
	return nextSection
}

func svgCharts(output *strings.Builder, charts []svgChart, y int) int {
	if len(charts) == 0 {
		return y
	}
	fmt.Fprintf(output, `<text x="24" y="%d" class="h2">CHARTS</text>`, y)
	y += 22
	left, right := balanceCharts(charts)
	leftY, rightY := y, y
	for _, chart := range left {
		chart.writeEmbedded(output, 24, leftY)
		leftY += chart.height + 16
	}
	for _, chart := range right {
		chart.writeEmbedded(output, 756, rightY)
		rightY += chart.height + 16
	}
	if rightY > leftY {
		leftY = rightY
	}
	return leftY + 14
}

func clip(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum-3]) + "..."
}
