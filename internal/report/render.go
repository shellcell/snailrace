package report

import (
	"time"

	"github.com/shellcell/snailrace/internal/model"
)

// Renderer caches report analysis and is intended for sequential rendering.
// Callers must not mutate the source report while the renderer is in use.
type Renderer struct {
	rawReport     model.Report
	displayReport model.Report
	metadata      ReportMetadata
	ranking       rankingData
	dimensionText string
	caveats       []string
	rawCaveats    []string
	chartGroups   []chartGroup
	charts        []svgChart
	chartsReady   bool
	comparisons   map[comparisonKey]deltaResult
	baselineRuns  map[metricID]map[int]float64
}

type ReportMetadata struct {
	MeasuredAt time.Time
	ToolNames  []string
}

type comparisonKey struct {
	candidate int
	metric    metricID
}

func NewRenderer(input model.Report) *Renderer {
	display := safeDisplayReport(input)
	ranking := calculateRanking(input)
	dimensions := includedDimensionsFromRanking(ranking)
	toolNames := make([]string, len(input.Benchmarks))
	for index, benchmark := range input.Benchmarks {
		toolNames[index] = benchmark.Tool.Name
	}
	return &Renderer{
		rawReport: input, displayReport: display, ranking: ranking,
		metadata:      ReportMetadata{MeasuredAt: input.MeasuredAt, ToolNames: toolNames},
		dimensionText: dimensionsText(dimensions),
		caveats:       reliabilityCaveats(display), rawCaveats: reliabilityCaveats(input),
		chartGroups: chartGroups(display), comparisons: make(map[comparisonKey]deltaResult),
		baselineRuns: make(map[metricID]map[int]float64),
	}
}

func (renderer *Renderer) Metadata() ReportMetadata {
	metadata := renderer.metadata
	metadata.ToolNames = append([]string(nil), metadata.ToolNames...)
	return metadata
}

func (renderer *Renderer) comparison(candidate int, metric metricID) deltaResult {
	key := comparisonKey{candidate: candidate, metric: metric}
	if result, ok := renderer.comparisons[key]; ok {
		return result
	}
	baseline := renderer.displayReport.Benchmarks[baselineIndex(renderer.displayReport)]
	row := metricCatalog[metric]
	baselineRuns, ok := renderer.baselineRuns[metric]
	if !ok {
		baselineRuns = indexMetricValues(baseline.Runs, row)
		renderer.baselineRuns[metric] = baselineRuns
	}
	result := compareMetricWithBaselineIndex(
		baseline, baselineRuns, renderer.displayReport.Benchmarks[candidate], row,
		renderer.displayReport.Config.IntervalMS/1000,
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
