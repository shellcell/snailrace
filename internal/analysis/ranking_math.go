package analysis

import (
	"math"
	"sort"

	"github.com/shellcell/snailrace/internal/model"
)

func normalized(values []float64, eligible []bool) []float64 {
	minimum := math.Inf(1)
	hasZero := false
	for index, value := range values {
		if !eligible[index] {
			continue
		}
		hasZero = hasZero || value == 0
		if value > 0 && value < minimum {
			minimum = value
		}
	}
	result := make([]float64, len(values))
	for index, value := range values {
		switch {
		case !eligible[index]:
			result[index] = math.Inf(1)
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

func normalizedPair(first, second []float64, eligible []bool, available bool) []float64 {
	result := make([]float64, len(first))
	if !available {
		for index := range result {
			result[index] = math.Inf(1)
		}
		return result
	}
	one, two := normalized(first, eligible), normalized(second, eligible)
	for index := range result {
		result[index] = geometricMean([]float64{one[index], two[index]})
	}
	return normalized(result, eligible)
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

func samplesReliable(config model.Config, benchmarks []model.Benchmark, eligible []bool) bool {
	minimum := config.IntervalMS / 1000 * 2
	if minimum <= 0 {
		return true
	}
	for index, benchmark := range benchmarks {
		if !eligible[index] {
			continue
		}
		for _, run := range benchmark.Runs {
			if run.WallSeconds < minimum || run.SampleCount < 2 ||
				run.SampleCoverageSeconds < config.IntervalMS/1000 {
				return false
			}
		}
	}
	return true
}

func hasPositive(values []float64, eligible []bool) bool {
	for index, value := range values {
		if eligible[index] && value > 0 {
			return true
		}
	}
	return false
}

func allPositive(values []float64, eligible []bool) bool {
	found := false
	for index, value := range values {
		if !eligible[index] {
			continue
		}
		found = true
		if value <= 0 || math.IsInf(value, 0) || math.IsNaN(value) {
			return false
		}
	}
	return found
}

func ranks(values []float64, eligible []bool) []int {
	type item struct {
		index int
		value float64
	}
	items := make([]item, 0, len(values))
	for index, value := range values {
		if eligible[index] && !math.IsInf(value, 0) && !math.IsNaN(value) {
			items = append(items, item{index: index, value: value})
		}
	}
	sort.SliceStable(items, func(left, right int) bool {
		return items[left].value < items[right].value
	})
	result := make([]int, len(values))
	rank := 0
	for position, current := range items {
		if position == 0 || current.value != items[position-1].value {
			rank = position + 1
		}
		result[current.index] = rank
	}
	return result
}
