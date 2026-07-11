package report

import (
	"fmt"
	"html"
	"math"
	"strings"

	"perftool/internal/model"
)

type rankingChartMetric struct {
	name   string
	format func(float64) string
	value  func(rankingRow) float64
	rank   func(rankingRow) int
}

func rankingCharts(report model.Report) []svgChart {
	ranking := calculateRanking(report)
	metrics := []rankingChartMetric{
		{"BALANCED INDEX", formatScore,
			func(row rankingRow) float64 { return row.overallScore },
			func(row rankingRow) int { return row.overallRank }},
		{ranking.primaryLabel, ranking.primaryUnit,
			func(row rankingRow) float64 { return row.primaryValue },
			func(row rankingRow) int { return row.primaryRank }},
	}
	if !(report.Config.Mode == "tui" && report.Config.DurationSeconds > 0) {
		metrics = append(metrics, rankingChartMetric{
			"CPU COST", formatDuration,
			func(row rankingRow) float64 { return row.cpuValue },
			func(row rankingRow) int { return row.cpuRank },
		})
	}
	if ranking.ramAvailable {
		metrics = append(metrics, rankingChartMetric{
			"RAM AGGREGATE", formatBytes,
			func(row rankingRow) float64 { return row.ramValue },
			func(row rankingRow) int { return row.ramRank },
		})
	}
	metrics = append(metrics, rankingChartMetric{
		"LINKED SIZE", formatBytes,
		func(row rankingRow) float64 { return row.footprintValue },
		func(row rankingRow) int { return row.footprintRank },
	})
	charts := make([]svgChart, 0, len(metrics))
	for _, metric := range metrics {
		charts = append(charts, rankingBarChart(report, ranking, metric))
	}
	return charts
}

func rankingBarChart(
	report model.Report,
	ranking rankingData,
	metric rankingChartMetric,
) svgChart {
	const left, plotWidth, rowHeight = 205, 355, 30
	maximum := 0.0
	for _, row := range ranking.rows {
		maximum = math.Max(maximum, metric.value(row))
	}
	if maximum <= 0 {
		maximum = 1
	}
	height := 62 + len(ranking.rows)*rowHeight
	var body strings.Builder
	body.WriteString(svgChartStyle)
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="23" class="title">RANKING · %s</text>`+
			`<text x="16" y="41" class="subtitle">descriptive point estimates · outline = category leader</text>`,
		html.EscapeString(metric.name),
	)
	for index, row := range ranking.rows {
		y := 62 + index*rowHeight
		value := metric.value(row)
		width := value / maximum * plotWidth
		color := toolColor(report, row.benchmark)
		outline := "none"
		outlineWidth := 0
		if metric.rank(row) == 1 {
			outline = "#a3be8c"
			outlineWidth = 2
		}
		fmt.Fprintf(
			&body,
			`<text x="16" y="%d" class="label" style="fill:%s">#%d %s</text>`+
				`<rect x="%d" y="%d" width="%.1f" height="13" rx="2" `+
				`fill="%s" stroke="%s" stroke-width="%d"/>`+
				`<text x="575" y="%d" class="value">%s</text>`,
			y, color, metric.rank(row),
			html.EscapeString(clip(reportToolLabel(report, row.benchmark), 26)),
			left, y-11, width, color, outline, outlineWidth, y, metric.format(value),
		)
	}
	return svgChart{
		kind: "ranking", title: metric.name, slug: chartSlug(metric.name),
		body: body.String(), height: height,
	}
}
