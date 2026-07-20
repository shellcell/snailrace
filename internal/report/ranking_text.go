package report

import (
	"fmt"
	"io"
	"math"

	"github.com/shellcell/snailrace/internal/model"
)

func writeTextRankingWith(writer io.Writer, report model.Report, ranking rankingData) {
	if len(ranking.Rows) == 0 {
		fmt.Fprintf(writer, "Ranking unavailable\t%s\n\n", ranking.UnavailableReason)
		return
	}
	fmt.Fprintln(writer, "Category leaders (point estimates)\tTool\tValue")
	for _, winner := range ranking.winners {
		fmt.Fprintf(
			writer, "%s\t%s\t%s\n", winner.category,
			winnerNames(report, winner, true), winner.value,
		)
	}
	fmt.Fprintln(writer)
	fmt.Fprintf(
		writer, "Comparison baseline\t%s\n\n",
		report.Benchmarks[baselineIndex(report)].Tool.Name,
	)
	if !ranking.Available {
		fmt.Fprintf(writer, "Overall ranking unavailable\t%s\n\n", ranking.UnavailableReason)
		return
	}
	fmt.Fprintf(
		writer,
		"Overall ranking (point estimates)\tTool\tBalanced\t%s\tCPU cost\tRAM aggregate\tLinked size\n",
		ranking.primaryLabel,
	)
	for _, row := range ranking.Rows {
		ramValue := "N/A"
		if ranking.RAMAvailable {
			ramValue = rankingCell(
				formatBytes(row.RAMValue), row.RAMScore, row.RAMRank, true,
			)
		}
		fmt.Fprintf(
			writer, "#%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.OverallRank, reportToolLabel(report, row.Benchmark),
			rankingCell(
				formatScore(row.OverallScore), row.OverallScore/ranking.BestOverall,
				row.OverallRank, true,
			),
			rankingCell(
				ranking.primaryUnit(row.PrimaryValue), row.PrimaryScore,
				row.PrimaryRank, ranking.PrimaryRatio,
			),
			rankingCell(
				cpuRankingValue(report, row), row.CPUScore, row.CPURank, ranking.CPURatio,
			),
			ramValue,
			rankingCell(
				formatBytes(row.FootprintValue), row.FootprintScore,
				row.FootprintRank, ranking.FootprintRatio,
			),
		)
	}
	fmt.Fprintln(writer)
}

func rankingCell(value string, score float64, rank int, ratioAvailable bool) string {
	if math.IsInf(score, 1) || math.IsNaN(score) {
		return "N/A"
	}
	if !ratioAvailable {
		return fmt.Sprintf("%s · Δ n/a (#%d)", value, rank)
	}
	return fmt.Sprintf(
		"%s · %s (#%d)", value, formatSignedPercent((score-1)*100), rank,
	)
}

func cpuRankingValue(report model.Report, row rankingRow) string {
	if report.Config.FixedDurationTUI() {
		return formatPercent(row.CPUValue)
	}
	return formatDuration(row.CPUValue)
}

func formatScore(value float64) string {
	if math.IsInf(value, 1) || math.IsNaN(value) {
		return "N/A"
	}
	return fmt.Sprintf("%.3fx", value)
}
