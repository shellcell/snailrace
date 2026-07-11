package report

import (
	"fmt"

	"perftool/internal/model"
)

func rankingWinners(report model.Report, ranking rankingData) []categoryWinner {
	if len(ranking.rows) == 0 {
		return nil
	}
	winners := []categoryWinner{
		winnerForRank(report, ranking.rows, "OVERALL", func(row rankingRow) int {
			return row.overallRank
		}, func(row rankingRow) string {
			return fmt.Sprintf("%.3fx balanced index", row.overallScore)
		}),
		winnerForRank(report, ranking.rows, ranking.primaryLabel, func(row rankingRow) int {
			return row.primaryRank
		}, func(row rankingRow) string {
			return ranking.primaryUnit(row.primaryValue)
		}),
		winnerForRank(report, ranking.rows, "CPU COST", func(row rankingRow) int {
			return row.cpuRank
		}, func(row rankingRow) string {
			if report.Config.Mode == "tui" && report.Config.DurationSeconds > 0 {
				return formatPercent(row.cpuValue)
			}
			return formatDuration(row.cpuValue)
		}),
	}
	if ranking.ramAvailable {
		winners = append(winners, winnerForRank(
			report, ranking.rows, "RAM COST",
			func(row rankingRow) int { return row.ramRank },
			func(row rankingRow) string { return formatBytes(row.ramValue) + " aggregate" },
		))
	}
	winners = append(winners, winnerForRank(
		report, ranking.rows, "LINKED SIZE",
		func(row rankingRow) int { return row.footprintRank },
		func(row rankingRow) string { return formatBytes(row.footprintValue) },
	))
	return winners
}

func winnerForRank(
	report model.Report,
	rows []rankingRow,
	category string,
	rank func(rankingRow) int,
	value func(rankingRow) string,
) categoryWinner {
	for _, row := range rows {
		if rank(row) == 1 {
			return categoryWinner{
				category: category, tool: report.Benchmarks[row.benchmark].Tool.Name,
				value: value(row),
			}
		}
	}
	return categoryWinner{category: category}
}
