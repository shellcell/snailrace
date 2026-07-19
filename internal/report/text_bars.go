package report

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/shellcell/snailrace/internal/model"
)

const compactBarWidth = 20

// writeCompactText renders the short, bar-chart stdout report used by default.
func writeCompactText(writer io.Writer, report model.Report) error {
	terminal := writerIsTerminal(writer)
	var body strings.Builder
	body.WriteString("\n")
	writeTextFailures(&body, report)
	if len(report.Benchmarks) == 1 {
		writeCompactSingle(&body, report, terminal)
	} else if len(report.Benchmarks) > 1 {
		writeCompactBars(&body, report, terminal)
	}
	_, err := io.WriteString(writer, body.String())
	return err
}

// compactMetric describes one bar group: a per-tool value with optional σ.
type compactMetric struct {
	name      string
	note      string
	unit      func(float64) string
	available bool
	value     func(rankingRow) float64
	stdDev    func(model.Summary) float64
}

func compactMetrics(report model.Report, ranking rankingData) []compactMetric {
	fixedTUI := report.Config.Mode == "tui" && report.Config.DurationSeconds > 0
	primaryStdDev := func(s model.Summary) float64 { return s.WallSeconds.StdDev }
	cpuStdDev := func(s model.Summary) float64 { return s.CPUTotalSeconds.StdDev }
	if fixedTUI {
		primaryStdDev = func(s model.Summary) float64 { return s.AverageCPUPercent.StdDev }
		cpuStdDev = primaryStdDev
	}
	ramNote := ""
	if ranking.ramPresent && !ranking.ramAvailable {
		ramNote = "  (sampling-limited · excluded from balanced index"
		if ranking.samplingInterval != "" {
			ramNote += "; lower -interval, now " + ranking.samplingInterval
		}
		ramNote += ")"
	}
	diskNote := ""
	for _, benchmark := range report.Benchmarks {
		if len(benchmark.Tool.SharedCacheFiles) > 0 {
			diskNote = "  (dyld shared cache excluded)"
			break
		}
	}
	return []compactMetric{
		{
			name: ranking.primaryLabel, unit: ranking.primaryUnit,
			available: ranking.primaryRatio,
			value:     func(r rankingRow) float64 { return r.primaryValue },
			stdDev:    primaryStdDev,
		},
		{
			name: "CPU", unit: func(v float64) string { return cpuCompactUnit(fixedTUI, v) },
			available: ranking.cpuRatio,
			value:     func(r rankingRow) float64 { return r.cpuValue },
			stdDev:    cpuStdDev,
		},
		{
			name: "RAM", note: ramNote, unit: formatBytes, available: ranking.ramPresent,
			value:  func(r rankingRow) float64 { return r.ramValue },
			stdDev: func(s model.Summary) float64 { return s.MeanResidentBytes.StdDev },
		},
		{
			name: "PHYS", unit: formatBytes, available: report.Host.OS == "darwin",
			value: func(r rankingRow) float64 {
				return report.Benchmarks[r.benchmark].Summary.PhysicalFootprintStats().Mean
			},
			stdDev: func(s model.Summary) float64 { return s.PhysicalFootprintStats().StdDev },
		},
		{
			name: "DISK", note: diskNote, unit: formatBytes, available: ranking.footprintRatio,
			value: func(r rankingRow) float64 { return r.footprintValue },
		},
	}
}

func cpuCompactUnit(fixedTUI bool, value float64) string {
	if fixedTUI {
		return formatPercent(value)
	}
	return formatDuration(value)
}

func writeCompactBars(body *strings.Builder, report model.Report, terminal bool) {
	ranking := calculateRanking(report)
	if len(ranking.rows) == 0 {
		fmt.Fprintf(body, "BALANCED  unavailable: %s\n\n", ranking.unavailableReason)
		return
	}
	rows := rowsByBenchmark(ranking.rows)
	labelWidth := compactLabelWidth(report)

	if ranking.available {
		winner := report.Benchmarks[ranking.rows[0].benchmark].Tool.Name
		fmt.Fprintf(
			body, "BALANCED  %s        winner %s\n",
			balancedIndexCategories(ranking), winner,
		)
		writeBarGroup(body, report, terminal, labelWidth, rows, func(r rankingRow) float64 {
			return r.overallScore
		}, func(r rankingRow) string { return formatScore(r.overallScore) })
	} else {
		fmt.Fprintf(body, "BALANCED  unavailable: %s\n\n", ranking.unavailableReason)
	}

	for _, metric := range compactMetrics(report, ranking) {
		if !metric.available {
			continue
		}
		fmt.Fprintf(body, "%s%s\n", metric.name, redNote(metric.note, terminal))
		writeBarGroup(body, report, terminal, labelWidth, rows, metric.value, func(r rankingRow) string {
			text := metric.unit(metric.value(r))
			if metric.stdDev != nil {
				sigma := metric.stdDev(report.Benchmarks[r.benchmark].Summary)
				text += " ± " + metric.unit(sigma)
			}
			return text
		})
	}
}

func writeBarGroup(
	body *strings.Builder,
	report model.Report,
	terminal bool,
	labelWidth int,
	rows map[int]rankingRow,
	value func(rankingRow) float64,
	annotate func(rankingRow) string,
) {
	maximum := 0.0
	order := make([]int, 0, len(report.Benchmarks))
	for index := range report.Benchmarks {
		if row, ok := rows[index]; ok {
			maximum = math.Max(maximum, value(row))
			order = append(order, index)
		}
	}
	// Best (lowest cost) first, so each group reads as a ranking.
	sort.SliceStable(order, func(i, j int) bool {
		return value(rows[order[i]]) < value(rows[order[j]])
	})
	for _, index := range order {
		row := rows[index]
		fraction := 0.0
		if maximum > 0 {
			fraction = value(row) / maximum
		}
		bar := textBar(fraction, compactBarWidth)
		label := padLabel(report.Benchmarks[index].Tool.Name, labelWidth)
		if terminal {
			color := terminalChartColor(index)
			bar = ansi(color, bar)
			label = ansi(color, label)
		}
		fmt.Fprintf(body, "  %d %s %s  %s\n", index+1, label, bar, annotate(row))
	}
	body.WriteString("\n")
}

func writeCompactSingle(body *strings.Builder, report model.Report, terminal bool) {
	ranking := calculateRanking(report)
	if len(ranking.rows) == 0 {
		return
	}
	row := ranking.rows[0]
	for _, metric := range compactMetrics(report, ranking) {
		if !metric.available {
			continue
		}
		text := metric.unit(metric.value(row))
		if metric.stdDev != nil {
			sigma := metric.stdDev(report.Benchmarks[row.benchmark].Summary)
			text += " ± " + metric.unit(sigma)
		}
		fmt.Fprintf(body, "  %-5s %s%s\n", metric.name, text, redNote(metric.note, terminal))
	}
	body.WriteString("\n")
}

// redNote colors a metric remark red on a terminal and leaves it plain otherwise.
func redNote(note string, terminal bool) string {
	if note == "" || !terminal {
		return note
	}
	return ansi("191;97;106", note)
}

func textBar(fraction float64, width int) string {
	if fraction < 0 || math.IsNaN(fraction) {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(math.Round(fraction * float64(width)))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func rowsByBenchmark(rows []rankingRow) map[int]rankingRow {
	result := make(map[int]rankingRow, len(rows))
	for _, row := range rows {
		result[row.benchmark] = row
	}
	return result
}

func compactLabelWidth(report model.Report) int {
	width := 0
	for index := range report.Benchmarks {
		if count := utf8.RuneCountInString(report.Benchmarks[index].Tool.Name); count > width {
			width = count
		}
	}
	return width
}

func padLabel(label string, width int) string {
	if pad := width - utf8.RuneCountInString(label); pad > 0 {
		return label + strings.Repeat(" ", pad)
	}
	return label
}
