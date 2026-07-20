package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shellcell/snailrace/internal/report"
)

func saveReportFormats(
	stderr io.Writer,
	directory string,
	formats []string,
	renderer *report.Renderer,
) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	formats = uniqueFormats(formats)
	needsCharts := false
	includeCommandCharts := false
	for _, format := range formats {
		info, _ := report.LookupFormat(format)
		needsCharts = needsCharts || info.NeedsCharts
		includeCommandCharts = includeCommandCharts || info.Name == report.FormatSVG
	}
	metadata := renderer.Metadata()
	stem, release, err := reserveReportStem(
		directory, reportStemFor(metadata.MeasuredAt, metadata.ToolNames),
	)
	if err != nil {
		return err
	}
	defer release()
	staging, err := os.MkdirTemp(directory, "."+stem+".tmp-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	var charts []report.ChartArtifact
	stagedBundle := filepath.Join(staging, stem)
	finalBundle := filepath.Join(directory, stem)
	if needsCharts {
		chartDirectory := filepath.Join(stagedBundle, "charts")
		charts, err = renderer.WriteChartFiles(
			chartDirectory, includeCommandCharts,
		)
		if err != nil {
			return err
		}
	}
	type stagedOutput struct{ staged, final string }
	var outputs []stagedOutput
	var announcements []string
	for _, format := range formats {
		switch format {
		case "svg":
			announcements = append(announcements, filepath.Join(finalBundle, "charts"))
		case "markdown":
			path := filepath.Join(stagedBundle, "report.md")
			if err := writeMarkdownFile(path, renderer, charts); err != nil {
				return err
			}
			announcements = append(announcements, filepath.Join(finalBundle, "report.md"))
		default:
			stagedPath := reportPathWithStem(staging, format, stem)
			if err := writeReportFile(stagedPath, format, renderer); err != nil {
				return err
			}
			finalPath := reportPathWithStem(directory, format, stem)
			outputs = append(outputs, stagedOutput{staged: stagedPath, final: finalPath})
			announcements = append(announcements, finalPath)
		}
	}
	if needsCharts {
		outputs = append(outputs, stagedOutput{staged: stagedBundle, final: finalBundle})
	}
	var published []string
	for _, output := range outputs {
		if err := renameNoReplace(output.staged, output.final); err != nil {
			for _, path := range published {
				os.RemoveAll(path)
			}
			return err
		}
		published = append(published, output.final)
	}
	for _, path := range announcements {
		if err := announce(stderr, path); err != nil {
			return err
		}
	}
	return nil
}

const maxReportNameAttempts = 10000

func reserveReportStem(
	directory string,
	base string,
) (string, func(), error) {
	for suffix := 1; suffix <= maxReportNameAttempts; suffix++ {
		stem := base
		if suffix > 1 {
			stem = fmt.Sprintf("%s-%d", base, suffix)
		}
		release, reserved, err := tryReserveStem(directory, stem)
		if err != nil {
			return "", nil, err
		}
		if reserved {
			return stem, release, nil
		}
	}
	return "", nil, fmt.Errorf("no available report name for %q in %s", base, directory)
}

func tryReserveStem(directory, stem string) (func(), bool, error) {
	lockPath := filepath.Join(directory, "."+stem+".lock")
	for attempt := range 2 {
		lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if errors.Is(err, os.ErrExist) {
			// Reclaim a lock leaked by a crashed run, then retry this stem once.
			if attempt == 0 && staleLockFile(lockPath) && os.Remove(lockPath) == nil {
				continue
			}
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		if err := lock.Close(); err != nil {
			os.Remove(lockPath)
			return nil, false, err
		}
		exists, err := reportStemExists(directory, stem)
		if err != nil || exists {
			os.Remove(lockPath)
			return nil, false, err
		}
		return func() { os.Remove(lockPath) }, true, nil
	}
	return nil, false, nil
}

// Locks are held only while a report is being written, so anything old enough
// to predate the current run by an hour was leaked by a crashed process.
func staleLockFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && time.Since(info.ModTime()) > time.Hour
}

func reportStemExists(directory, stem string) (bool, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Name() == stem || strings.HasPrefix(entry.Name(), stem+".") {
			return true, nil
		}
	}
	return false, nil
}

func writeMarkdownFile(
	path string,
	renderer *report.Renderer,
	charts []report.ChartArtifact,
) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := renderer.WriteMarkdownWithCharts(file, charts, "charts")
	syncErr := error(nil)
	if writeErr == nil {
		syncErr = file.Sync()
	}
	closeErr := file.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}

func writeReportFile(path, format string, renderer *report.Renderer) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := renderer.Write(file, format)
	syncErr := error(nil)
	if writeErr == nil {
		syncErr = file.Sync()
	}
	closeErr := file.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}

func uniqueFormats(formats []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(formats))
	for _, format := range formats {
		info, ok := report.LookupFormat(format)
		if !ok {
			format = strings.ToLower(strings.TrimSpace(format))
		} else {
			format = string(info.Name)
		}
		if !seen[format] {
			seen[format] = true
			result = append(result, format)
		}
	}
	return result
}

func announce(writer io.Writer, path string) error {
	_, err := fmt.Fprintf(writer, "Report saved to %s\n", path)
	return err
}
