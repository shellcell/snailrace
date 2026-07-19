package report

import (
	"github.com/shellcell/snailrace/internal/analysis"
	"github.com/shellcell/snailrace/internal/model"
)

func calculateRanking(report model.Report) rankingData {
	numeric := analysis.Calculate(report.Config, report.Benchmarks)
	result := rankingData{
		Ranking: numeric,
	}
	if result.RAMPresent && !result.RAMAvailable {
		result.samplingInterval = currentIntervalLabel(report.Config)
	}
	result.primaryLabel, result.primaryUnit = "TIME", formatDuration
	if result.FixedTUI {
		result.primaryLabel, result.primaryUnit = "CPU", formatPercent
	}
	result.winners = rankingWinners(report, result)
	return result
}
