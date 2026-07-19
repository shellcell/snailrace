package report

import (
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

func fullCommand(benchmark model.Benchmark) string {
	return strings.Join(benchmark.Tool.Command, " ")
}
