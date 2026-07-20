package report

import "github.com/shellcell/snailrace/internal/analysis"

type rankingRow = analysis.RankingRow

type categoryWinner struct {
	category   string
	benchmarks []int
	value      string
}

type rankingData struct {
	analysis.Ranking
	winners          []categoryWinner
	primaryLabel     string
	primaryUnit      func(float64) string
	samplingInterval string
}
