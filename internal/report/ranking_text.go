package report

import (
	"fmt"
	"io"
	"math"

	"perftool/internal/model"
)

func writeTextRanking(writer io.Writer, report model.Report) {
	ranking := calculateRanking(report)
	fmt.Fprintln(writer, "Category winners\tTool\tValue")
	baselineName := report.Benchmarks[baselineIndex(report)].Tool.Name
	for _, winner := range ranking.winners {
		name := winner.tool
		if name == baselineName {
			name += " [BASELINE]"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", winner.category, name, winner.value)
	}
	fmt.Fprintln(writer)
	fmt.Fprintf(
		writer, "Comparison baseline\t%s\n\n",
		report.Benchmarks[baselineIndex(report)].Tool.Name,
	)
	fmt.Fprintf(
		writer,
		"Overall ranking\tTool\tBalanced\t%s\tCPU cost\tRAM aggregate\tLinked size\n",
		ranking.primaryLabel,
	)
	for _, row := range ranking.rows {
		ramValue := "N/A"
		if ranking.ramAvailable {
			ramValue = rankingCell(
				formatBytes(row.ramValue), row.ramScore, row.ramRank,
			)
		}
		fmt.Fprintf(
			writer, "#%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.overallRank, reportToolLabel(report, row.benchmark),
			rankingCell(
				formatScore(row.overallScore), row.overallScore/ranking.bestOverall,
				row.overallRank,
			),
			rankingCell(
				ranking.primaryUnit(row.primaryValue), row.primaryScore,
				row.primaryRank,
			),
			rankingCell(cpuRankingValue(report, row), row.cpuScore, row.cpuRank),
			ramValue,
			rankingCell(
				formatBytes(row.footprintValue), row.footprintScore,
				row.footprintRank,
			),
		)
	}
	fmt.Fprintln(writer)
}

func rankingCell(value string, score float64, rank int) string {
	if math.IsInf(score, 1) || math.IsNaN(score) {
		return "N/A"
	}
	return fmt.Sprintf(
		"%s · %s (#%d)", value, formatSignedPercent((score-1)*100), rank,
	)
}

func cpuRankingValue(report model.Report, row rankingRow) string {
	if report.Config.Mode == "tui" && report.Config.DurationSeconds > 0 {
		return formatPercent(row.cpuValue)
	}
	return formatDuration(row.cpuValue)
}

func formatScore(value float64) string {
	if math.IsInf(value, 1) || math.IsNaN(value) {
		return "N/A"
	}
	return fmt.Sprintf("%.3fx", value)
}
