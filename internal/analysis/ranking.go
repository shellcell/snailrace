package analysis

import (
	"math"
	"sort"

	"perftool/internal/model"
)

type Ranking struct {
	Rows           []RankingRow
	RAMAvailable   bool
	BestOverall    float64
	FixedTUI       bool
	PrimaryRatio   bool
	CPURatio       bool
	FootprintRatio bool
	IndexPrimary   bool
	IndexCPU       bool
	IndexRAM       bool
	IndexFootprint bool
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
	result.RAMAvailable = result.RAMAvailable && allPositive(meanRAM) && allPositive(peakRAM)
	result.PrimaryRatio = allPositive(primary)
	result.CPURatio = allPositive(cpu)
	result.FootprintRatio = allPositive(footprint)
	primaryScore, cpuScore := normalized(primary), normalized(cpu)
	ramScore := normalizedPair(meanRAM, peakRAM, result.RAMAvailable)
	footprintScore := normalized(footprint)
	included := indexSet(config.IndexDimensions)
	result.IndexPrimary = included["time"] && !result.FixedTUI && result.PrimaryRatio
	result.IndexCPU = (included["cpu"] || (result.FixedTUI && included["time"])) &&
		result.CPURatio
	result.IndexRAM = included["ram"] && result.RAMAvailable
	result.IndexFootprint = included["disk"] && result.FootprintRatio
	for index := range result.Rows {
		var scores []float64
		if result.IndexPrimary {
			scores = append(scores, primaryScore[index])
		}
		if result.IndexCPU {
			scores = append(scores, cpuScore[index])
		}
		if result.IndexRAM {
			scores = append(scores, ramScore[index])
		}
		if result.IndexFootprint {
			scores = append(scores, footprintScore[index])
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

// IndexDimensions are the cost categories that may compose the balanced index.
var IndexDimensions = []string{"time", "cpu", "ram", "disk"}

// DefaultIndexDimensions is the balanced index used when none is configured.
var DefaultIndexDimensions = []string{"time", "cpu", "ram"}

// ValidIndexDimension reports whether token names a known index dimension.
func ValidIndexDimension(token string) bool {
	for _, dimension := range IndexDimensions {
		if token == dimension {
			return true
		}
	}
	return false
}

func indexSet(dimensions []string) map[string]bool {
	if len(dimensions) == 0 {
		dimensions = DefaultIndexDimensions
	}
	set := make(map[string]bool, len(dimensions))
	for _, dimension := range dimensions {
		set[dimension] = true
	}
	return set
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
