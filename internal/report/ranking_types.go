package report

type rankingRow struct {
	benchmark      int
	overallRank    int
	primaryRank    int
	cpuRank        int
	ramRank        int
	footprintRank  int
	overallScore   float64
	primaryScore   float64
	cpuScore       float64
	ramScore       float64
	footprintScore float64
	primaryValue   float64
	cpuValue       float64
	ramValue       float64
	footprintValue float64
}

type categoryWinner struct {
	category   string
	benchmarks []int
	value      string
}

type rankingData struct {
	rows           []rankingRow
	winners        []categoryWinner
	primaryLabel   string
	primaryUnit    func(float64) string
	ramAvailable   bool
	primaryRatio   bool
	cpuRatio       bool
	footprintRatio bool
	bestOverall    float64
}
