package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"perftool/internal/model"
	"perftool/internal/report"
)

func saveReportFormats(
	stderr io.Writer,
	directory string,
	formats []string,
	result model.Report,
) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	formats = uniqueFormats(formats)
	needsCharts := containsFormat(formats, "svg") ||
		containsFormat(formats, "markdown") || containsFormat(formats, "md")
	var charts []report.ChartArtifact
	bundleDirectory := filepath.Join(directory, reportStem(result))
	if needsCharts {
		chartDirectory := filepath.Join(bundleDirectory, "charts")
		var err error
		charts, err = report.WriteChartFiles(
			chartDirectory, result, containsFormat(formats, "svg"),
		)
		if err != nil {
			return err
		}
	}
	for _, format := range formats {
		switch format {
		case "svg":
			if err := announce(stderr, filepath.Join(bundleDirectory, "charts")); err != nil {
				return err
			}
		case "markdown", "md":
			path := filepath.Join(bundleDirectory, "report.md")
			if err := writeMarkdownFile(path, result, charts); err != nil {
				return err
			}
			if err := announce(stderr, path); err != nil {
				return err
			}
		default:
			path := reportPath(directory, format, result)
			if err := writeReportFile(path, format, result); err != nil {
				return err
			}
			if err := announce(stderr, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeMarkdownFile(
	path string,
	result model.Report,
	charts []report.ChartArtifact,
) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := report.WriteMarkdownWithCharts(file, result, charts, "charts")
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func writeReportFile(path, format string, result model.Report) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := report.Write(file, format, result)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func uniqueFormats(formats []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(formats))
	for _, format := range formats {
		if !seen[format] {
			seen[format] = true
			result = append(result, format)
		}
	}
	return result
}

func containsFormat(formats []string, target string) bool {
	for _, format := range formats {
		if format == target {
			return true
		}
	}
	return false
}

func announce(writer io.Writer, path string) error {
	_, err := fmt.Fprintf(writer, "Report saved to %s\n", path)
	return err
}
