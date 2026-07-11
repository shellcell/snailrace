package report

import (
	"fmt"
	"html"

	"perftool/internal/model"
)

func htmlToolLabel(report model.Report, index int, badge bool) string {
	label := fmt.Sprintf(
		`<span style="color:%s">%s</span>`,
		toolColor(report, index),
		html.EscapeString(report.Benchmarks[index].Tool.Name),
	)
	if badge && index == baselineIndex(report) {
		label += ` <span class="badge">BASELINE</span>`
	}
	return label
}

func benchmarkIndexByName(report model.Report, name string) int {
	for index, benchmark := range report.Benchmarks {
		if benchmark.Tool.Name == name {
			return index
		}
	}
	return 0
}
