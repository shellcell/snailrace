package report

import (
	"encoding/json"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

type jsonWinner struct {
	Category string `json:"category"`
	Tool     string `json:"tool"`
	Value    string `json:"formatted_value"`
}

type jsonFailure struct {
	Tool      string `json:"tool"`
	Failed    int    `json:"failed_runs"`
	Total     int    `json:"total_runs"`
	ExitCodes []int  `json:"exit_codes"`
}

type jsonRankingRow struct {
	Rank                  int      `json:"rank,omitempty"`
	Tool                  string   `json:"tool"`
	BalancedIndex         *float64 `json:"balanced_index,omitempty"`
	BalancedDeltaPercent  *float64 `json:"balanced_delta_percent,omitempty"`
	PrimaryValue          float64  `json:"primary_value"`
	PrimaryRank           int      `json:"primary_rank"`
	PrimaryDeltaPercent   *float64 `json:"primary_delta_percent,omitempty"`
	CPUValue              float64  `json:"cpu_value"`
	CPURank               int      `json:"cpu_rank"`
	CPUDeltaPercent       *float64 `json:"cpu_delta_percent,omitempty"`
	RAMValueBytes         float64  `json:"ram_value_bytes,omitempty"`
	RAMRank               int      `json:"ram_rank,omitempty"`
	RAMDeltaPercent       *float64 `json:"ram_delta_percent,omitempty"`
	FootprintBytes        float64  `json:"footprint_bytes"`
	FootprintRank         int      `json:"footprint_rank"`
	FootprintDeltaPercent *float64 `json:"footprint_delta_percent,omitempty"`
}

type jsonRanking struct {
	Available          bool             `json:"available"`
	UnavailableReason  string           `json:"unavailable_reason,omitempty"`
	Method             string           `json:"method"`
	PrimaryMetric      string           `json:"primary_metric"`
	IncludedDimensions []string         `json:"included_dimensions"`
	Winners            []jsonWinner     `json:"winners"`
	Rows               []jsonRankingRow `json:"rows"`
}

func writeJSON(writer io.Writer, renderer *Renderer) error {
	report := renderer.rawReport
	payload := struct {
		model.Report
		Ranking     jsonRanking   `json:"ranking"`
		Failures    []jsonFailure `json:"failures,omitempty"`
		Reliability []string      `json:"reliability_caveats,omitempty"`
	}{
		Report: report, Ranking: makeJSONRanking(report, renderer.ranking),
		Failures: makeJSONFailures(report), Reliability: renderer.rawCaveats,
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func makeJSONFailures(report model.Report) []jsonFailure {
	failures := benchmarkFailures(report)
	result := make([]jsonFailure, 0, len(failures))
	for _, failure := range failures {
		result = append(result, jsonFailure{
			Tool:   report.Benchmarks[failure.benchmark].Tool.Name,
			Failed: failure.failed, Total: failure.total,
			ExitCodes: append([]int(nil), failure.exitCodes...),
		})
	}
	return result
}

func makeJSONRanking(report model.Report, ranking rankingData) jsonRanking {
	result := jsonRanking{
		Available: ranking.Available, UnavailableReason: ranking.UnavailableReason,
		Method: "equal-weight geometric mean of normalized category costs; included: " +
			balancedIndexCategories(ranking),
		PrimaryMetric:      ranking.primaryLabel,
		IncludedDimensions: append([]string(nil), includedDimensionsFromRanking(ranking)...),
	}
	for _, winner := range ranking.winners {
		result.Winners = append(result.Winners, jsonWinner{
			Category: winner.category, Tool: winnerNames(report, winner, false),
			Value: winner.value,
		})
	}
	for _, row := range ranking.Rows {
		item := jsonRankingRow{
			Tool:         report.Benchmarks[row.Benchmark].Tool.Name,
			PrimaryValue: row.PrimaryValue, PrimaryRank: row.PrimaryRank,
			CPUValue: row.CPUValue, CPURank: row.CPURank,
			FootprintBytes: row.FootprintValue, FootprintRank: row.FootprintRank,
		}
		if ranking.Available {
			item.Rank = row.OverallRank
			item.BalancedIndex = floatPointer(row.OverallScore)
			item.BalancedDeltaPercent = floatPointer(
				(row.OverallScore/ranking.BestOverall - 1) * 100,
			)
		}
		if ranking.PrimaryRatio {
			item.PrimaryDeltaPercent = floatPointer((row.PrimaryScore - 1) * 100)
		}
		if ranking.CPURatio {
			item.CPUDeltaPercent = floatPointer((row.CPUScore - 1) * 100)
		}
		if ranking.FootprintRatio {
			item.FootprintDeltaPercent = floatPointer((row.FootprintScore - 1) * 100)
		}
		if ranking.RAMAvailable {
			item.RAMValueBytes, item.RAMRank = row.RAMValue, row.RAMRank
			item.RAMDeltaPercent = floatPointer((row.RAMScore - 1) * 100)
		}
		result.Rows = append(result.Rows, item)
	}
	return result
}

func floatPointer(value float64) *float64 { return &value }
