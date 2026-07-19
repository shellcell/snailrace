package report

import (
	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/style"
)

func fullCommand(benchmark model.Benchmark) string {
	command := benchmark.Tool.Command
	if len(command) == 0 {
		return ""
	}
	return style.CommandText(command, benchmark.Tool.ShellCommand)
}
