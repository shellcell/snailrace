package report

import (
	"fmt"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

func rankingWinners(report model.Report, ranking rankingData) []categoryWinner {
	if len(ranking.Rows) == 0 {
		return nil
	}
	var winners []categoryWinner
	appendWinner := func(winner categoryWinner) {
		if len(winner.benchmarks) > 0 {
			winners = append(winners, winner)
		}
	}
	if ranking.Available {
		appendWinner(winnerForRank(ranking.Rows, "OVERALL", func(row rankingRow) int {
			return row.OverallRank
		}, func(row rankingRow) string {
			return fmt.Sprintf("%.3fx balanced index", row.OverallScore)
		}))
	}
	appendWinner(winnerForRank(ranking.Rows, ranking.primaryLabel, func(row rankingRow) int {
		return row.PrimaryRank
	}, func(row rankingRow) string {
		return ranking.primaryUnit(row.PrimaryValue)
	}))
	if !report.Config.FixedDurationTUI() {
		appendWinner(winnerForRank(
			ranking.Rows, "CPU COST", func(row rankingRow) int {
				return row.CPURank
			}, func(row rankingRow) string {
				return formatDuration(row.CPUValue)
			}))
	}
	if ranking.RAMAvailable {
		appendWinner(winnerForRank(
			ranking.Rows, "RAM COST",
			func(row rankingRow) int { return row.RAMRank },
			func(row rankingRow) string { return formatBytes(row.RAMValue) + " aggregate" },
		))
	}
	appendWinner(winnerForRank(
		ranking.Rows, "LINKED SIZE",
		func(row rankingRow) int { return row.FootprintRank },
		func(row rankingRow) string { return formatBytes(row.FootprintValue) },
	))
	return winners
}

func winnerForRank(
	rows []rankingRow,
	category string,
	rank func(rankingRow) int,
	value func(rankingRow) string,
) categoryWinner {
	var benchmarks []int
	for _, row := range rows {
		if rank(row) == 1 {
			benchmarks = append(benchmarks, row.Benchmark)
		}
	}
	result := categoryWinner{category: category, benchmarks: benchmarks}
	if len(benchmarks) > 0 {
		for _, row := range rows {
			if row.Benchmark == benchmarks[0] {
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
