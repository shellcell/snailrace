package report

import (
	"fmt"
	"io"
	"strings"

	"perftool/internal/model"
)

func Write(writer io.Writer, format string, report model.Report) error {
	switch strings.ToLower(format) {
	case "text", "txt":
		return writeText(writer, report)
	case "json":
		return writeJSON(writer, report)
	case "markdown", "md":
		return writeMarkdown(writer, report)
	case "html":
		return writeHTML(writer, report)
	case "svg":
		return writeSVG(writer, report)
	default:
		return fmt.Errorf("unknown report format %q", format)
	}
}

func ValidFormat(format string) bool {
	switch strings.ToLower(format) {
	case "text", "txt", "json", "markdown", "md", "html", "svg":
		return true
	default:
		return false
	}
}
