package report

import "perftool/internal/model"

func reportToolLabel(report model.Report, index int) string {
	label := report.Benchmarks[index].Tool.Name
	if index == baselineIndex(report) {
		label += " [BASELINE]"
	}
	return label
}
