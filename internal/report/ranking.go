package report

import (
	"github.com/shellcell/snailrace/internal/analysis"
	"github.com/shellcell/snailrace/internal/model"
)

func calculateRanking(report model.Report) rankingData {
	numeric := analysis.Calculate(report.Config, report.Benchmarks)
	result := rankingData{
		rows:      make([]rankingRow, len(numeric.Rows)),
		available: numeric.Available, unavailableReason: numeric.UnavailableReason,
		ramAvailable: numeric.RAMAvailable, ramPresent: numeric.RAMPresent,
		bestOverall:  numeric.BestOverall,
		primaryRatio: numeric.PrimaryRatio, cpuRatio: numeric.CPURatio,
		footprintRatio: numeric.FootprintRatio,
		indexPrimary:   numeric.IndexPrimary, indexCPU: numeric.IndexCPU,
		indexRAM: numeric.IndexRAM, indexFootprint: numeric.IndexFootprint,
	}
	if result.ramPresent && !result.ramAvailable {
		result.samplingInterval = currentIntervalLabel(report.Config)
	}
	for index, row := range numeric.Rows {
		result.rows[index] = rankingRow{
			benchmark: row.Benchmark, overallRank: row.OverallRank,
			primaryRank: row.PrimaryRank, cpuRank: row.CPURank,
			ramRank: row.RAMRank, footprintRank: row.FootprintRank,
			overallScore: row.OverallScore, primaryScore: row.PrimaryScore,
			cpuScore: row.CPUScore, ramScore: row.RAMScore,
			footprintScore: row.FootprintScore, primaryValue: row.PrimaryValue,
			cpuValue: row.CPUValue, ramValue: row.RAMValue,
			footprintValue: row.FootprintValue,
		}
	}
	result.primaryLabel, result.primaryUnit = "TIME", formatDuration
	if numeric.FixedTUI {
		result.primaryLabel, result.primaryUnit = "CPU", formatPercent
	}
	result.winners = rankingWinners(report, result)
	return result
}
