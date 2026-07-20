package app

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/shellcell/snailrace/internal/runner"
	"github.com/shellcell/snailrace/internal/style"
)

// printCommandLegend lists each tool's label and full command before measurement
// begins, coloring labels with the tool palette when the writer is a terminal.
func printCommandLegend(writer io.Writer, specs []runner.Spec) {
	if len(specs) == 0 {
		return
	}
	colored := false
	if file, ok := writer.(interface{ Fd() uintptr }); ok {
		colored = term.IsTerminal(int(file.Fd()))
	}
	labelWidth := 0
	for _, spec := range specs {
		if width := utf8.RuneCountInString(spec.Name); width > labelWidth {
			labelWidth = width
		}
	}
	numberWidth := len(strconv.Itoa(len(specs)))
	for index, spec := range specs {
		label := spec.Name
		padding := strings.Repeat(" ", labelWidth-utf8.RuneCountInString(label))
		if colored {
			label = progressToolColor(index, label)
		}
		fmt.Fprintf(
			writer, "%*d. %s%s  %s\n",
			numberWidth, index+1, label, padding, specCommand(spec),
		)
	}
	fmt.Fprintln(writer)
}

func specCommand(spec runner.Spec) string {
	if spec.Shell != "" {
		return style.CommandText([]string{spec.Shell}, true)
	}
	return style.CommandText(spec.Args, false)
}
