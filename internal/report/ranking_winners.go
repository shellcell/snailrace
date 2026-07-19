package report

import (
	"fmt"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
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
	}
	if !(report.Config.Mode == "tui" && report.Config.DurationSeconds > 0) {
		winners = append(winners, winnerForRank(
			report, ranking.rows, "CPU COST", func(row rankingRow) int {
				return row.cpuRank
			}, func(row rankingRow) string {
				return formatDuration(row.cpuValue)
			}))
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
	var benchmarks []int
	for _, row := range rows {
		if rank(row) == 1 {
			benchmarks = append(benchmarks, row.benchmark)
		}
	}
	result := categoryWinner{category: category, benchmarks: benchmarks}
	if len(benchmarks) > 0 {
		for _, row := range rows {
			if row.benchmark == benchmarks[0] {
				result.value = value(row)
				break
			}
		}
	}
	return result
}

func winnerNames(report model.Report, winner categoryWinner, baseline bool) string {
	names := make([]string, 0, len(winner.benchmarks))
	for _, index := range winner.benchmarks {
		name := report.Benchmarks[index].Tool.Name
		if baseline && index == baselineIndex(report) {
			name += " [BASELINE]"
		}
		names = append(names, name)
	}
	return strings.Join(names, " / ")
}
