package report

import (
	"fmt"
	"html"
	"math"
	"strings"
	"unicode"

	"perftool/internal/model"
	"perftool/internal/style"
)

const chartWidth = 720

const svgChartStyle = `<style>` +
	`.title{fill:#eceff4;font-size:13px;font-weight:700}` +
	`.subtitle{fill:#d8dee9;font-size:10px}.label{fill:#e5e9f0;font-size:11px}` +
	`.value{fill:#d8dee9;font-size:10px}.axis{stroke:#4c566a;stroke-width:1}` +
	`</style>`

type svgChart struct {
	kind        string
	title       string
	slug        string
	description string
	body        string
	height      int
}

type chartMetric struct {
	name   string
	format func(float64) string
	stats  func(model.Benchmark) model.Stats
	run    func(model.Run) float64
	row    int
}

type chartGroup struct {
	name    string
	metrics []chartMetric
}

func (chart svgChart) html() string {
	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" role="img" `+
			`viewBox="0 0 %d %d" width="100%%" `+
			`style="max-width:%dpx;font-family:monospace">%s%s</svg>`,
		chartWidth, chart.height, chartWidth, chart.metadata(), chart.body,
	)
}

func (chart svgChart) embedded(x, y int) string {
	return fmt.Sprintf(
		`<svg x="%d" y="%d" width="%d" height="%d" `+
			`viewBox="0 0 %d %d">%s%s</svg>`,
		x, y, chartWidth, chart.height, chartWidth, chart.height,
		chart.metadata(), chart.body,
	)
}

func (chart svgChart) metadata() string {
	return fmt.Sprintf(
		`<title>%s</title><desc>%s</desc>`,
		html.EscapeString(chart.accessibleTitle()), html.EscapeString(chart.explanation()),
	)
}

func (chart svgChart) accessibleTitle() string {
	if chart.title != "" {
		return chart.title
	}
	return "Snailrace chart"
}

func (chart svgChart) explanation() string {
	if chart.description != "" {
		return chart.description
	}
	return chart.accessibleTitle()
}

func chartColor(index, _ int) string {
	return style.Tool(index).Hex
}

func toolColor(_ model.Report, index int) string {
	return style.Tool(index).Hex
}

func statusColor(class string) string {
	switch class {
	case "good":
		return "#a3be8c"
	case "bad":
		return "#bf616a"
	case "uncertain":
		return "#ebcb8b"
	default:
		return "#81a1c1"
	}
}

func niceDeltaScale(value float64) float64 {
	value = math.Max(1, value)
	power := math.Pow(10, math.Floor(math.Log10(value)))
	normalized := value / power
	switch {
	case normalized <= 1:
		return power
	case normalized <= 2:
		return 2 * power
	case normalized <= 5:
		return 5 * power
	default:
		return 10 * power
	}
}

func balanceCharts(charts []svgChart) ([]svgChart, []svgChart) {
	var left, right []svgChart
	leftHeight, rightHeight := 0, 0
	for _, chart := range charts {
		if leftHeight <= rightHeight {
			left = append(left, chart)
			leftHeight += chart.height
		} else {
			right = append(right, chart)
			rightHeight += chart.height
		}
	}
	return left, right
}

func chartSlug(value string) string {
	var result strings.Builder
	lastDash := false
	for _, character := range strings.ToLower(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			result.WriteRune(character)
			lastDash = false
		} else if !lastDash && result.Len() > 0 {
			result.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(result.String(), "-")
}
