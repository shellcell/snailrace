package app

import (
	"fmt"

	"perftool/internal/style"
)

func progressToolColor(index int, value string) string {
	color := style.Tool(index)
	return fmt.Sprintf(
		"\x1b[38;2;%d;%d;%dm%s\x1b[0m",
		color.R, color.G, color.B, value,
	)
}
