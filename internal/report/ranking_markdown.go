package report

import (
	"fmt"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

func writeMarkdownRanking(writer io.Writer, report model.Report) {
	ranking := calculateRanking(report)
	if len(ranking.rows) == 0 {
		fmt.Fprintf(
			writer, "## Ranking Unavailable\n\n%s.\n\n",
			escapeMarkdown(ranking.unavailableReason),
		)
		return
	}
	fmt.Fprintln(writer, "## Category Leaders (Point Estimates)")
	fmt.Fprintln(writer, "\n| Category | Tool | Value |")
	fmt.Fprintln(writer, "|---|---|---:|")
	for _, winner := range ranking.winners {
		fmt.Fprintf(
			writer, "| %s | %s | %s |\n", winner.category,
			escapeMarkdown(winnerNames(report, winner, true)), winner.value,
		)
	}
	fmt.Fprintf(
		writer, "\nComparison baseline: **%s**. Lower balanced index is better.\n",
		escapeMarkdown(report.Benchmarks[baselineIndex(report)].Tool.Name),
	)
	if !ranking.available {
		fmt.Fprintf(
			writer, "\n## Overall Ranking Unavailable\n\n%s.\n\n",
			escapeMarkdown(ranking.unavailableReason),
		)
		return
	}
	fmt.Fprintln(writer, "\n## Overall Ranking (Point Estimates)")
	fmt.Fprintf(
		writer,
		"\n| Rank | Tool | Balanced | %s | CPU cost | RAM aggregate | Linked size |\n",
		ranking.primaryLabel,
	)
	fmt.Fprintln(writer, "|---:|---|---:|---:|---:|---:|---:|")
	for _, row := range ranking.rows {
		ramValue := "N/A"
		if ranking.ramAvailable {
			ramValue = rankingCell(
				formatBytes(row.ramValue), row.ramScore, row.ramRank, true,
			)
		}
		fmt.Fprintf(
			writer, "| #%d | %s | %s | %s | %s | %s | %s |\n",
			row.overallRank,
			escapeMarkdown(reportToolLabel(report, row.benchmark)),
			rankingCell(
				formatScore(row.overallScore), row.overallScore/ranking.bestOverall,
				row.overallRank, true,
			),
			rankingCell(
				ranking.primaryUnit(row.primaryValue), row.primaryScore,
				row.primaryRank, ranking.primaryRatio,
			),
			rankingCell(
				cpuRankingValue(report, row), row.cpuScore, row.cpuRank, ranking.cpuRatio,
			),
			ramValue,
			rankingCell(
				formatBytes(row.footprintValue), row.footprintScore,
				row.footprintRank, ranking.footprintRatio,
			),
		)
	}
	fmt.Fprintln(writer)
}
