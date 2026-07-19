package report

import (
	"fmt"
	"html"
	"math"
	"strings"
)

type forestRow struct {
	label            string
	point, low, high float64
	status, class    string
	identity         string
	interval         bool
	baseline         bool
}

func deltaForestChart(renderer *Renderer, group chartGroup) svgChart {
	report := renderer.displayReport
	baselinePosition := baselineIndex(report)
	baseline := report.Benchmarks[baselinePosition]
	rows := make([]forestRow, 0)
	maximum := 0.0
	for _, metric := range group.metrics {
		row := metric
		if !availableFor(row, report.Host.OS, baseline.Summary) {
			rows = append(rows, forestRow{
				label: metric.chartName, status: "N/A on " + report.Host.OS,
				class: "neutral", identity: toolColor(baselinePosition),
				baseline: true,
			})
			continue
		}
		baseStats := metric.stats(baseline.Summary)
		baseMean := baseStats.Mean
		if baseStats.N == 0 || !finiteNumber(baseMean) {
			rows = append(rows, forestRow{
				label: metric.chartName, status: "N/A", class: "neutral",
				identity: toolColor(baselinePosition), baseline: true,
			})
			continue
		}
		baselineRow := forestRow{
			label:  metric.chartName + " · " + reportToolLabel(report, baselinePosition),
			status: metric.format(baseMean) + " · baseline",
			class:  "neutral", identity: toolColor(baselinePosition),
			baseline: true,
		}
		if baseMean != 0 && baseStats.CI95Valid {
			baselineRow.low, baselineRow.high = percentInterval(baseStats.CI95Low, baseStats.CI95High, baseMean)
			baselineRow.interval = finiteNumber(baselineRow.low) &&
				finiteNumber(baselineRow.high)
			if baselineRow.interval {
				maximum = math.Max(
					maximum, math.Max(math.Abs(baselineRow.low), math.Abs(baselineRow.high)),
				)
			}
		}
		rows = append(rows, baselineRow)
		if baseMean == 0 {
			for index, candidate := range report.Benchmarks {
				if index == baselinePosition {
					continue
				}
				if !availableFor(row, report.Host.OS, candidate.Summary) {
					rows = append(rows, forestRow{
						label:  metric.chartName + " · " + candidate.Tool.Name,
						status: "N/A", class: "neutral", identity: toolColor(index),
					})
					continue
				}
				value := metric.stats(candidate.Summary).Mean
				delta := renderer.comparison(index, metric.id)
				rows = append(rows, forestRow{
					label:  metric.chartName + " · " + candidate.Tool.Name,
					status: metric.format(value) + " · Δ% n/a · " + delta.status,
					class:  delta.class, identity: toolColor(index),
					baseline: true,
				})
			}
			continue
		}
		for index, candidate := range report.Benchmarks {
			if index == baselinePosition {
				continue
			}
			if !availableFor(row, report.Host.OS, candidate.Summary) {
				rows = append(rows, forestRow{
					label:  metric.chartName + " · " + candidate.Tool.Name,
					status: "N/A", class: "neutral", identity: toolColor(index),
				})
				continue
			}
			delta := renderer.comparison(index, metric.id)
			low, high := 0.0, 0.0
			interval := delta.difference.CI95Valid
			if interval {
				low = delta.difference.CI95Low / baseMean * 100
				high = delta.difference.CI95High / baseMean * 100
				interval = finiteNumber(low) && finiteNumber(high)
			}
			if interval {
				maximum = math.Max(maximum, math.Max(math.Abs(low), math.Abs(high)))
			}
			if delta.percentAvailable {
				maximum = math.Max(maximum, math.Abs(delta.percent))
			}
			rows = append(rows, forestRow{
				label: metric.chartName + " · " + candidate.Tool.Name,
				point: delta.percent, low: low, high: high,
				status: delta.status, class: delta.class,
				identity: toolColor(index), interval: interval,
			})
		}
	}
	if len(rows) == 0 {
		return svgChart{}
	}
	maximum = niceDeltaScale(maximum)
	return renderForest(group.name, baseline.Tool.Name, rows, maximum)
}

func renderForest(
	title, baseline string,
	rows []forestRow,
	maximum float64,
) svgChart {
	const left, plotWidth, rowHeight = 225, 330, 30
	center := float64(left) + plotWidth/2
	height := 76 + len(rows)*rowHeight
	position := func(value float64) float64 {
		value = math.Max(-maximum, math.Min(maximum, value))
		return center + value/maximum*(plotWidth/2)
	}
	var body strings.Builder
	body.WriteString(svgChartStyle)
	fmt.Fprintf(
		&body,
		`<rect width="100%%" height="100%%" rx="8" fill="#3b4252"/>`+
			`<text x="16" y="24" class="title">Δ FROM BASELINE · %s</text>`+
			`<text x="16" y="43" class="subtitle">baseline: %s (Δ 0%%) · `+
			`candidate whisker = paired 95%% CI · baseline whisker = mean 95%% CI</text>`+
			`<path d="M %.1f 54 V %d" stroke="#4c566a" stroke-dasharray="3 3"/>`+
			`<text x="%d" y="62" class="value">-%g%%</text>`+
			`<text x="%.1f" y="62" text-anchor="middle" class="value">0</text>`+
			`<text x="%d" y="62" text-anchor="end" class="value">+%g%%</text>`,
		html.EscapeString(title), html.EscapeString(clip(baseline, 40)),
		center, height-12, left, maximum,
		center, left+plotWidth, maximum,
	)
	for index, row := range rows {
		y := 72 + index*rowHeight
		color := statusColor(row.class)
		if row.baseline {
			whisker := ""
			if row.interval {
				whisker = fmt.Sprintf(
					`<path class="baseline-ci" d="M %.1f %d H %.1f M %.1f %d v 10 M %.1f %d v 10" `+
						`stroke="%s" stroke-width="2"/>`,
					position(row.low), y-4, position(row.high), position(row.low), y-9,
					position(row.high), y-9, color,
				)
			}
			fmt.Fprintf(
				&body,
				`<text x="16" y="%d" class="label" style="fill:%s">%s</text>`+
					`%s`+
					`<path d="M %.1f %d l 6 6 -6 6 -6 -6 z" fill="%s"/>`+
					`<text x="570" y="%d" fill="%s" font-size="11">%s</text>`,
				y, row.identity, html.EscapeString(clip(row.label, 31)), whisker, position(0), y-10,
				color, y, color, html.EscapeString(row.status),
			)
			continue
		}
		if !row.interval {
			fmt.Fprintf(
				&body,
				`<text x="16" y="%d" class="label" style="fill:%s">%s</text>`+
					`<circle cx="%.1f" cy="%d" r="5" fill="%s"/>`+
					`<text x="570" y="%d" fill="%s" font-size="11">%s %s</text>`,
				y, row.identity, html.EscapeString(clip(row.label, 31)),
				position(row.point), y-4, color, y, color,
				formatSignedPercent(row.point), row.status,
			)
			continue
		}
		fmt.Fprintf(
			&body,
			`<text x="16" y="%d" class="label" style="fill:%s">%s</text>`+
				`<path d="M %.1f %d H %.1f M %.1f %d v 10 M %.1f %d v 10" `+
				`stroke="%s" stroke-width="2"/>`+
				`<circle cx="%.1f" cy="%d" r="5" fill="%s"/>`+
				`<text x="570" y="%d" fill="%s" font-size="11">%s %s</text>`,
			y, row.identity, html.EscapeString(clip(row.label, 31)), position(row.low), y-4,
			position(row.high), position(row.low), y-9, position(row.high), y-9,
			color, position(row.point), y-4, color, y, color,
			formatSignedPercent(row.point), row.status,
		)
	}
	return svgChart{
		kind: "baseline", title: title, slug: chartSlug(title),
		description: baselineChartDescription(title, baseline),
		body:        body.String(), height: height,
	}
}

func percentInterval(low, high, center float64) (float64, float64) {
	lowPercent := (low/center - 1) * 100
	highPercent := (high/center - 1) * 100
	if lowPercent > highPercent {
		return highPercent, lowPercent
	}
	return lowPercent, highPercent
}

func baselineChartDescription(title, baseline string) string {
	return "Shows each candidate's percent change in " + title + " relative to baseline " +
		baseline + ". Math: percent delta = (candidate mean / baseline mean - 1) * 100. " +
		"Candidate whiskers are paired, pointwise 95% confidence intervals for run " +
		"differences, scaled by the baseline mean. The baseline whisker is the baseline " +
		"mean 95% confidence interval expressed relative to the baseline mean."
}
