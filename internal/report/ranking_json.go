package report

import (
	"encoding/json"
	"io"

	"perftool/internal/model"
)

type jsonWinner struct {
	Category string `json:"category"`
	Tool     string `json:"tool"`
	Value    string `json:"formatted_value"`
}

type jsonRankingRow struct {
	Rank                  int     `json:"rank"`
	Tool                  string  `json:"tool"`
	BalancedIndex         float64 `json:"balanced_index"`
	BalancedDeltaPercent  float64 `json:"balanced_delta_percent"`
	PrimaryValue          float64 `json:"primary_value"`
	PrimaryRank           int     `json:"primary_rank"`
	PrimaryDeltaPercent   float64 `json:"primary_delta_percent"`
	CPUValue              float64 `json:"cpu_value"`
	CPURank               int     `json:"cpu_rank"`
	CPUDeltaPercent       float64 `json:"cpu_delta_percent"`
	RAMValueBytes         float64 `json:"ram_value_bytes,omitempty"`
	RAMRank               int     `json:"ram_rank,omitempty"`
	RAMDeltaPercent       float64 `json:"ram_delta_percent,omitempty"`
	FootprintBytes        float64 `json:"footprint_bytes"`
	FootprintRank         int     `json:"footprint_rank"`
	FootprintDeltaPercent float64 `json:"footprint_delta_percent"`
}

type jsonRanking struct {
	Method        string           `json:"method"`
	PrimaryMetric string           `json:"primary_metric"`
	Winners       []jsonWinner     `json:"winners"`
	Rows          []jsonRankingRow `json:"rows"`
}

func writeJSON(writer io.Writer, report model.Report) error {
	payload := struct {
		model.Report
		Ranking jsonRanking `json:"ranking"`
	}{Report: report, Ranking: makeJSONRanking(report)}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func makeJSONRanking(report model.Report) jsonRanking {
	ranking := calculateRanking(report)
	result := jsonRanking{
		Method:        "equal-weight geometric mean of available normalized category costs",
		PrimaryMetric: ranking.primaryLabel,
	}
	for _, winner := range ranking.winners {
		result.Winners = append(result.Winners, jsonWinner{
			Category: winner.category, Tool: winner.tool, Value: winner.value,
		})
	}
	for _, row := range ranking.rows {
		item := jsonRankingRow{
			Rank: row.overallRank, Tool: report.Benchmarks[row.benchmark].Tool.Name,
			BalancedIndex:        row.overallScore,
			BalancedDeltaPercent: (row.overallScore/ranking.bestOverall - 1) * 100,
			PrimaryValue:         row.primaryValue, PrimaryRank: row.primaryRank,
			PrimaryDeltaPercent: (row.primaryScore - 1) * 100,
			CPUValue:            row.cpuValue, CPURank: row.cpuRank,
			CPUDeltaPercent: (row.cpuScore - 1) * 100,
			FootprintBytes:  row.footprintValue, FootprintRank: row.footprintRank,
			FootprintDeltaPercent: (row.footprintScore - 1) * 100,
		}
		if ranking.ramAvailable {
			item.RAMValueBytes, item.RAMRank = row.ramValue, row.ramRank
			item.RAMDeltaPercent = (row.ramScore - 1) * 100
		}
		result.Rows = append(result.Rows, item)
	}
	return result
}
