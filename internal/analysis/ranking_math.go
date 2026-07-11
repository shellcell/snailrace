package analysis

import (
	"math"

	"perftool/internal/model"
)

func normalized(values []float64) []float64 {
	minimum := math.Inf(1)
	hasZero := false
	for _, value := range values {
		hasZero = hasZero || value == 0
		if value > 0 && value < minimum {
			minimum = value
		}
	}
	result := make([]float64, len(values))
	for index, value := range values {
		switch {
		case value == 0:
			result[index] = 1
		case math.IsInf(minimum, 1) || value < 0:
			result[index] = math.Inf(1)
		case hasZero:
			result[index] = 1 + value/minimum
		default:
			result[index] = value / minimum
		}
	}
	return result
}

func normalizedPair(first, second []float64, available bool) []float64 {
	result := make([]float64, len(first))
	if !available {
		for index := range result {
			result[index] = math.Inf(1)
		}
		return result
	}
	one, two := normalized(first), normalized(second)
	for index := range result {
		result[index] = geometricMean([]float64{one[index], two[index]})
	}
	return normalized(result)
}

func geometricMean(values []float64) float64 {
	total, count := 0.0, 0
	for _, value := range values {
		if value > 0 && !math.IsInf(value, 1) {
			total += math.Log(value)
			count++
		}
	}
	if count == 0 {
		return math.Inf(1)
	}
	return math.Exp(total / float64(count))
}

func samplesReliable(config model.Config, benchmarks []model.Benchmark) bool {
	minimum := config.IntervalMS / 1000 * 2
	if minimum <= 0 {
		return true
	}
	for _, benchmark := range benchmarks {
		for _, run := range benchmark.Runs {
			if run.WallSeconds < minimum || run.SampleCount < 2 ||
				run.SampleCoverageSeconds < config.IntervalMS/1000 {
				return false
			}
		}
	}
	return true
}

func hasPositive(values []float64) bool {
	for _, value := range values {
		if value > 0 {
			return true
		}
	}
	return false
}

func allPositive(values []float64) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value <= 0 || math.IsInf(value, 0) || math.IsNaN(value) {
			return false
		}
	}
	return true
}

func rankOf(values []float64, target int) int {
	rank := 1
	for index, value := range values {
		if index != target && value < values[target] {
			rank++
		}
	}
	return rank
}
