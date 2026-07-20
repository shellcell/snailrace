package report

import (
	"errors"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

var errWriteLimit = errors.New("write limit reached")

type limitedWriter struct {
	remaining int
}

func (writer *limitedWriter) Write(value []byte) (int, error) {
	if writer.remaining <= 0 {
		return 0, errWriteLimit
	}
	if len(value) > writer.remaining {
		written := writer.remaining
		writer.remaining = 0
		return written, errWriteLimit
	}
	writer.remaining -= len(value)
	return len(value), nil
}

func TestReportFormatsPropagateWriterErrors(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithWallTimes("tool", 1, 2)},
	}
	for _, format := range []string{"text", "html", "markdown", "svg", "json"} {
		t.Run(format, func(t *testing.T) {
			err := NewRenderer(report).Write(&limitedWriter{}, format)
			if !errors.Is(err, errWriteLimit) {
				t.Fatalf("error = %v, want write failure", err)
			}
		})
	}
}
