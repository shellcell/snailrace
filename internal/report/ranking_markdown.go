package report

import (
	"fmt"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

func writeMarkdownRankingWith(writer io.Writer, report model.Report, ranking rankingData) {
	if len(ranking.Rows) == 0 {
		fmt.Fprintf(
			writer, "## Ranking Unavailable\n\n%s.\n\n",
			escapeMarkdown(ranking.UnavailableReason),
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
	if !ranking.Available {
		fmt.Fprintf(
			writer, "\n## Overall Ranking Unavailable\n\n%s.\n\n",
			escapeMarkdown(ranking.UnavailableReason),
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
	for _, row := range ranking.Rows {
		ramValue := "N/A"
		if ranking.RAMAvailable {
			ramValue = rankingCell(
				formatBytes(row.RAMValue), row.RAMScore, row.RAMRank, true,
			)
		}
		fmt.Fprintf(
			writer, "| #%d | %s | %s | %s | %s | %s | %s |\n",
			row.OverallRank,
			escapeMarkdown(reportToolLabel(report, row.Benchmark)),
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
