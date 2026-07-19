package analysis

import (
	"math"
	"sort"

	"github.com/shellcell/snailrace/internal/model"
)

type Ranking struct {
	Rows              []RankingRow
	Available         bool
	UnavailableReason string
	RAMAvailable      bool
	RAMPresent        bool
	BestOverall       float64
	FixedTUI          bool
	PrimaryRatio      bool
	CPURatio          bool
	FootprintRatio    bool
	IndexPrimary      bool
	IndexCPU          bool
	IndexRAM          bool
	IndexFootprint    bool
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
	result := Ranking{}
	if count == 0 {
		result.UnavailableReason = "no benchmarks"
		return result
	}
	result.FixedTUI = config.Mode == "tui" && config.DurationSeconds > 0
	eligible := make([]bool, count)
	eligibleCount := 0
	primary, cpu := make([]float64, count), make([]float64, count)
	meanRAM, peakRAM := make([]float64, count), make([]float64, count)
	footprint := make([]float64, count)
	for index, benchmark := range benchmarks {
		eligible[index] = benchmark.EligibleForRanking()
		if eligible[index] {
			eligibleCount++
		}
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
	result.RAMAvailable = samplesReliable(config, benchmarks, eligible)
	if !hasPositive(meanRAM, eligible) || !hasPositive(peakRAM, eligible) {
		result.RAMAvailable = false
	}
	result.RAMPresent = allPositive(meanRAM, eligible) && allPositive(peakRAM, eligible)
	result.RAMAvailable = result.RAMAvailable && result.RAMPresent
	result.PrimaryRatio = allPositive(primary, eligible)
	result.CPURatio = allPositive(cpu, eligible)
	result.FootprintRatio = allPositive(footprint, eligible)
	primaryScore, cpuScore := normalized(primary, eligible), normalized(cpu, eligible)
	ramScore := normalizedPair(meanRAM, peakRAM, eligible, result.RAMAvailable)
	footprintScore := normalized(footprint, eligible)
	included := indexSet(config.IndexDimensions)
	result.IndexPrimary = included["time"] && !result.FixedTUI && result.PrimaryRatio
	result.IndexCPU = (included["cpu"] || (result.FixedTUI && included["time"])) &&
		result.CPURatio
	result.IndexRAM = included["ram"] && result.RAMAvailable
	result.IndexFootprint = included["disk"] && result.FootprintRatio
	result.Available = eligibleCount > 0 && (result.IndexPrimary || result.IndexCPU ||
		result.IndexRAM || result.IndexFootprint)
	if eligibleCount == 0 {
		result.UnavailableReason = "no successful measured runs"
	} else if !result.Available {
		result.UnavailableReason = "none of the selected dimensions has usable positive values"
	}
	result.Rows = make([]RankingRow, 0, eligibleCount)
	for index := range benchmarks {
		if !eligible[index] {
			continue
		}
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
		if result.RAMPresent {
			ramValue = geometricMean([]float64{meanRAM[index], peakRAM[index]})
		}
		overallScore := 0.0
		if result.Available {
			overallScore = geometricMean(scores)
		}
		result.Rows = append(result.Rows, RankingRow{
			Benchmark: index, OverallScore: overallScore,
			PrimaryScore: primaryScore[index], CPUScore: cpuScore[index],
			RAMScore: ramScore[index], FootprintScore: footprintScore[index],
			PrimaryValue: primary[index], CPUValue: cpu[index],
			RAMValue: ramValue, FootprintValue: footprint[index],
		})
	}
	applyRanks(
		result.Rows, eligible, result.Available,
		primaryScore, cpuScore, ramScore, footprintScore,
	)
	if result.Available {
		result.BestOverall = math.Inf(1)
		for _, row := range result.Rows {
			result.BestOverall = math.Min(result.BestOverall, row.OverallScore)
		}
	}
	return result
}

func AutomaticBaseline(config model.Config, benchmarks []model.Benchmark) (int, bool) {
	ranking := Calculate(config, benchmarks)
	if !ranking.Available || len(ranking.Rows) == 0 {
		return 0, false
	}
	return ranking.Rows[0].Benchmark + 1, true
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

func applyRanks(
	rows []RankingRow,
	eligible []bool,
	overallAvailable bool,
	primary, cpu, ram, footprint []float64,
) {
	overall := make([]float64, len(eligible))
	for index := range overall {
		overall[index] = math.Inf(1)
	}
	for _, row := range rows {
		overall[row.Benchmark] = row.OverallScore
	}
	for index := range rows {
		benchmark := rows[index].Benchmark
		if overallAvailable {
			rows[index].OverallRank = rankOf(overall, benchmark, eligible)
		}
		rows[index].PrimaryRank = rankOf(primary, benchmark, eligible)
		rows[index].CPURank = rankOf(cpu, benchmark, eligible)
		rows[index].RAMRank = rankOf(ram, benchmark, eligible)
		rows[index].FootprintRank = rankOf(footprint, benchmark, eligible)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if !overallAvailable {
			return rows[i].Benchmark < rows[j].Benchmark
		}
		return rows[i].OverallRank < rows[j].OverallRank
	})
}
