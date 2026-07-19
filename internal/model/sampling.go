package model

func SamplingReliable(benchmark Benchmark, intervalSeconds float64) bool {
	if len(benchmark.Runs) == 0 {
		return false
	}
	if intervalSeconds <= 0 {
		return true
	}
	minimumDuration := intervalSeconds * 2
	for _, run := range benchmark.Runs {
		if run.WallSeconds < minimumDuration || run.SampleCount < 2 ||
			run.SampleCoverageSeconds < intervalSeconds {
			return false
		}
	}
	return true
}
