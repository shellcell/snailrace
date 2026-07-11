package report

import (
	"fmt"
	"io"

	"perftool/internal/model"
)

func writeMarkdownRanking(writer io.Writer, report model.Report) {
	ranking := calculateRanking(report)
	baselineName := report.Benchmarks[baselineIndex(report)].Tool.Name
	fmt.Fprintln(writer, "## Category Winners")
	fmt.Fprintln(writer, "\n| Category | Tool | Value |")
	fmt.Fprintln(writer, "|---|---|---:|")
	for _, winner := range ranking.winners {
		name := winner.tool
		if name == baselineName {
			name += " [BASELINE]"
		}
		fmt.Fprintf(
			writer, "| %s | %s | %s |\n", winner.category,
			escapeMarkdown(name), winner.value,
		)
	}
	fmt.Fprintf(
		writer, "\nComparison baseline: **%s**. Lower balanced index is better.\n",
		escapeMarkdown(report.Benchmarks[baselineIndex(report)].Tool.Name),
	)
	fmt.Fprintln(writer, "\n## Overall Ranking")
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
				formatBytes(row.ramValue), row.ramScore, row.ramRank,
			)
		}
		fmt.Fprintf(
			writer, "| #%d | %s | %s | %s | %s | %s | %s |\n",
			row.overallRank,
			escapeMarkdown(reportToolLabel(report, row.benchmark)),
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
