package report

import "github.com/shellcell/snailrace/internal/model"

type Renderer struct {
	raw           model.Report
	report        model.Report
	ranking       rankingData
	dimensionText string
	caveats       []string
	rawCaveats    []string
	groups        []chartGroup
	charts        []svgChart
	chartsReady   bool
	comparisons   map[comparisonKey]deltaResult
}

type comparisonKey struct {
	candidate int
	metric    int
}

func NewRenderer(input model.Report) *Renderer {
	display := safeDisplayReport(input)
	ranking := calculateRanking(input)
	dimensions := includedDimensionsFromRanking(ranking)
	return &Renderer{
		raw: input, report: display, ranking: ranking,
		dimensionText: dimensionsText(dimensions),
		caveats:       reliabilityCaveats(display), rawCaveats: reliabilityCaveats(input),
		groups: chartGroups(display), comparisons: make(map[comparisonKey]deltaResult),
	}
}

func (renderer *Renderer) comparison(candidate, metric int) deltaResult {
	key := comparisonKey{candidate: candidate, metric: metric}
	if result, ok := renderer.comparisons[key]; ok {
		return result
	}
	baseline := renderer.report.Benchmarks[baselineIndex(renderer.report)]
	result := compareMetric(
		baseline, renderer.report.Benchmarks[candidate], metricRows[metric],
		renderer.report.Config.IntervalMS/1000,
	)
	renderer.comparisons[key] = result
	return result
}

func (renderer *Renderer) reportCharts() []svgChart {
	if !renderer.chartsReady {
		renderer.charts = buildReportCharts(renderer)
		renderer.chartsReady = true
	}
	return renderer.charts
}
