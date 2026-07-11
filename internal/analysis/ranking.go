package analysis

import (
	"math"
	"sort"

	"perftool/internal/model"
)

type Ranking struct {
	Rows         []RankingRow
	RAMAvailable bool
	BestOverall  float64
	FixedTUI     bool
}

type RankingRow struct {
	Benchmark      int
	OverallRank    int
	PrimaryRank    int
	CPURank        int
	RAMRank        int
	FootprintRank  int
	OverallScore   float64
	PrimaryScore   float64
	CPUScore       float64
	RAMScore       float64
	FootprintScore float64
	PrimaryValue   float64
	CPUValue       float64
	RAMValue       float64
	FootprintValue float64
}

func Calculate(config model.Config, benchmarks []model.Benchmark) Ranking {
	count := len(benchmarks)
	result := Ranking{Rows: make([]RankingRow, count)}
	if count == 0 {
		return result
	}
	result.FixedTUI = config.Mode == "tui" && config.DurationSeconds > 0
	result.RAMAvailable = samplesReliable(config, benchmarks)
	primary, cpu := make([]float64, count), make([]float64, count)
	meanRAM, peakRAM := make([]float64, count), make([]float64, count)
	footprint := make([]float64, count)
	for index, benchmark := range benchmarks {
		primary[index] = benchmark.Summary.WallSeconds.Mean
		cpu[index] = benchmark.Summary.CPUTotalSeconds.Mean
		if result.FixedTUI {
			primary[index] = benchmark.Summary.AverageCPUPercent.Mean
			cpu[index] = primary[index]
		}
		meanRAM[index] = benchmark.Summary.MeanResidentBytes.Mean
		peakRAM[index] = benchmark.Summary.PeakResidentBytes.Mean
		footprint[index] = float64(benchmark.Tool.DiskFootprintBytes)
	}
	if !hasPositive(meanRAM) || !hasPositive(peakRAM) {
		result.RAMAvailable = false
	}
	primaryScore, cpuScore := normalized(primary), normalized(cpu)
	ramScore := normalizedPair(meanRAM, peakRAM, result.RAMAvailable)
	footprintScore := normalized(footprint)
	for index := range result.Rows {
		scores := []float64{primaryScore[index], cpuScore[index], footprintScore[index]}
		if result.FixedTUI {
			scores = []float64{cpuScore[index], footprintScore[index]}
		}
		if result.RAMAvailable {
			scores = append(scores, ramScore[index])
		}
		ramValue := 0.0
		if result.RAMAvailable {
			ramValue = geometricMean([]float64{meanRAM[index], peakRAM[index]})
		}
		result.Rows[index] = RankingRow{
			Benchmark: index, OverallScore: geometricMean(scores),
			PrimaryScore: primaryScore[index], CPUScore: cpuScore[index],
			RAMScore: ramScore[index], FootprintScore: footprintScore[index],
			PrimaryValue: primary[index], CPUValue: cpu[index],
			RAMValue: ramValue, FootprintValue: footprint[index],
		}
	}
	applyRanks(result.Rows, primaryScore, cpuScore, ramScore, footprintScore)
	result.BestOverall = math.Inf(1)
	for _, row := range result.Rows {
		result.BestOverall = math.Min(result.BestOverall, row.OverallScore)
	}
	return result
}

func AutomaticBaseline(config model.Config, benchmarks []model.Benchmark) int {
	ranking := Calculate(config, benchmarks)
	if len(ranking.Rows) == 0 {
		return 1
	}
	return ranking.Rows[0].Benchmark + 1
}

func BalancedIndexes(config model.Config, benchmarks []model.Benchmark) []float64 {
	ranking := Calculate(config, benchmarks)
	indexes := make([]float64, len(benchmarks))
	for _, row := range ranking.Rows {
		indexes[row.Benchmark] = row.OverallScore
	}
	return indexes
}

func applyRanks(rows []RankingRow, primary, cpu, ram, footprint []float64) {
	overall := make([]float64, len(rows))
	for index := range rows {
		overall[index] = rows[index].OverallScore
	}
	for index := range rows {
		rows[index].OverallRank = rankOf(overall, index)
		rows[index].PrimaryRank = rankOf(primary, index)
		rows[index].CPURank = rankOf(cpu, index)
		rows[index].RAMRank = rankOf(ram, index)
		rows[index].FootprintRank = rankOf(footprint, index)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].OverallRank < rows[j].OverallRank
	})
}
