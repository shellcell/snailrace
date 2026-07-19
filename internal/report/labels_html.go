package report

import (
	"fmt"
	"html"

	"github.com/shellcell/snailrace/internal/model"
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
