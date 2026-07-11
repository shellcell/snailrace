package report

import (
	"strings"

	"perftool/internal/model"
)

func fullCommand(benchmark model.Benchmark) string {
	return strings.Join(benchmark.Tool.Command, " ")
}
