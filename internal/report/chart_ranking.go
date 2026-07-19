package report

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

type rankingChartMetric struct {
	name   string
	format func(float64) string
	value  func(rankingRow) float64
	rank   func(rankingRow) int
}

func rankingCharts(report model.Report) []svgChart {
	return rankingChartsWith(report, calculateRanking(report))
}

func rankingChartsWith(report model.Report, ranking rankingData) []svgChart {
	if len(ranking.rows) == 0 {
		return nil
	}
	var metrics []rankingChartMetric
	if ranking.available {
		metrics = append(metrics, rankingChartMetric{"BALANCED INDEX", formatScore,
			func(row rankingRow) float64 { return row.overallScore },
			func(row rankingRow) int { return row.overallRank }})
	}
	metrics = append(metrics, rankingChartMetric{ranking.primaryLabel, ranking.primaryUnit,
		func(row rankingRow) float64 { return row.primaryValue },
		func(row rankingRow) int { return row.primaryRank }})
	if !(report.Config.Mode == "tui" && report.Config.DurationSeconds > 0) {
		metrics = append(metrics, rankingChartMetric{
			"CPU COST", formatDuration,
			func(row rankingRow) float64 { return row.cpuValue },
			func(row rankingRow) int { return row.cpuRank },
		})
	}
	if ranking.ramPresent {
		ramRank := func(row rankingRow) int { return row.ramRank }
		if !ranking.ramAvailable {
			// Sampling-limited RAM has no reliable score; rank descriptively by value.
			order := rankByValue(ranking.rows, func(row rankingRow) float64 { return row.ramValue })
			ramRank = func(row rankingRow) int { return order[row.benchmark] }
		}
		metrics = append(metrics, rankingChartMetric{
			"RAM AGGREGATE", formatBytes,
			func(row rankingRow) float64 { return row.ramValue },
			ramRank,
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

func rankByValue(rows []rankingRow, value func(rankingRow) float64) map[int]int {
	result := make(map[int]int, len(rows))
	ordered := append([]rankingRow(nil), rows...)
	sort.SliceStable(ordered, func(left, right int) bool {
		return value(ordered[left]) < value(ordered[right])
	})
	rank := 0
	for position, row := range ordered {
		if position == 0 || value(row) != value(ordered[position-1]) {
			rank = position + 1
		}
		result[row.benchmark] = rank
	}
	return result
}

func rankingBarChart(
	report model.Report,
	ranking rankingData,
	metric rankingChartMetric,
) svgChart {
	const left, plotWidth, rowHeight = 205, 355, 30
	maximum := 0.0
	for _, row := range ranking.rows {
		value := metric.value(row)
		if !finiteChartValue(value) || value < 0 {
			return svgChart{}
		}
		maximum = math.Max(maximum, value)
	}
	if maximum <= 0 {
		maximum = 1
	}
	height := 62 + len(ranking.rows)*rowHeight
	var body strings.Builder
	body.WriteString(svgChartStyle)
	subtitle := "descriptive point estimates · outline = category leader"
	if metric.name == "RAM AGGREGATE" && ranking.ramPresent && !ranking.ramAvailable {
		subtitle = "sampling-limited · descriptive only · excluded from balanced index"
		if ranking.samplingInterval != "" {
			subtitle += " · lower -interval (now " + ranking.samplingInterval + ")"
		}
	}
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="23" class="title">RANKING · %s</text>`+
			`<text x="16" y="41" class="subtitle">%s</text>`,
		html.EscapeString(metric.name), html.EscapeString(subtitle),
	)
	rows := append([]rankingRow(nil), ranking.rows...)
	sort.SliceStable(rows, func(left, right int) bool {
		return metric.rank(rows[left]) < metric.rank(rows[right])
	})
	for index, row := range rows {
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
		description: rankingChartDescription(metric.name, ranking),
		body:        body.String(), height: height,
	}
}

func rankingChartDescription(metric string, ranking rankingData) string {
	switch metric {
	case "BALANCED INDEX":
		return "Ranks tools by balanced index, the geometric mean of included normalized " +
			"cost scores. Included here: " + balancedIndexCategories(ranking) +
			". For positive categories, score = value / best value; categories with " +
			"nonpositive or unavailable values are omitted. RAM combines normalized mean RSS " +
			"and peak RSS with a geometric mean. Lower is better; this is descriptive."
	case "TIME":
		return "Ranks tools by mean wall-clock time. Math: mean = sum(run wall time) / n. " +
			"Lower bars are better; outlines mark category leaders. " +
			"This ranking does not include uncertainty intervals."
	case "CPU", "CPU COST":
		return "Ranks tools by CPU cost. Math: command mode value = mean(user CPU + system CPU); " +
			"fixed-duration TUI value = mean average CPU percent. Lower bars are better; " +
			"this ranking does not include uncertainty intervals."
	case "RAM AGGREGATE":
		caveat := ""
		if ranking.ramPresent && !ranking.ramAvailable {
			caveat = " These values are sampling-limited (short runs or too few valid " +
				"samples), so they are shown for reference but excluded from the balanced index."
			if ranking.samplingInterval != "" {
				caveat += " The current sampling interval is " + ranking.samplingInterval +
					"; re-run with a smaller -interval for reliable RAM."
			}
		}
		return "Ranks tools by RAM aggregate. Math: value = sqrt(mean of per-run mean RSS " +
			"* mean of per-run peak RSS), using sampled tree RSS. Lower bars are better; " +
			"this ranking does not include uncertainty intervals." + caveat
	case "LINKED SIZE":
		return "Ranks tools by linked disk footprint. Math: value = executable bytes + " +
			"linked-library bytes discovered before measurement. Lower bars are better; " +
			"this ranking has no runtime uncertainty interval."
	}
	return "Ranks tools by the " + strings.ToLower(metric) +
		" point estimate. Lower bars are better for this cost chart; outlines mark " +
		"category leaders. This ranking does not include uncertainty intervals."
}

func balancedIndexCategories(ranking rankingData) string {
	var categories []string
	if ranking.indexPrimary {
		categories = append(categories, primaryCategoryName(ranking.primaryLabel))
	}
	if ranking.indexCPU {
		categories = append(categories, "CPU cost")
	}
	if ranking.indexRAM {
		categories = append(categories, "RAM aggregate")
	}
	if ranking.indexFootprint {
		categories = append(categories, "linked size")
	}
	if len(categories) == 0 {
		return "none"
	}
	return strings.Join(categories, ", ")
}

func primaryCategoryName(label string) string {
	if label == "TIME" {
		return "wall time"
	}
	return strings.ToLower(label)
}
