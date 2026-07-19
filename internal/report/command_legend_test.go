package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

func TestFullCommandQuotesDirectArgumentsFaithfully(t *testing.T) {
	benchmark := model.Benchmark{Tool: model.ToolInfo{Command: []string{
		"tool", "two words", "it's", "", "line\nbreak",
	}}}
	want := `tool 'two words' 'it'"'"'s' '' "line\nbreak"`
	if got := fullCommand(benchmark); got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestFullCommandQuotesSingleDirectArgument(t *testing.T) {
	benchmark := model.Benchmark{Tool: model.ToolInfo{Command: []string{"/tmp/my tool"}}}
	if got, want := fullCommand(benchmark), `'/tmp/my tool'`; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestRenderedLabelsContainNoTerminalControls(t *testing.T) {
	run := model.Run{WallSeconds: 1, StopReason: "exited"}
	report := model.Report{Verbose: true, Config: model.Config{Baseline: 1},
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "safe\n\x1b[31munsafe", Command: []string{"true"}},
			Runs: []model.Run{run}, Summary: model.Summarize([]model.Run{run}),
		}}}
	for _, format := range []string{"text", "markdown", "html", "svg"} {
		var output bytes.Buffer
		if err := NewRenderer(report).Write(&output, format); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if strings.ContainsAny(output.String(), "\n\x1b") &&
			strings.Contains(output.String(), "safe\n\x1b") {
			t.Fatalf("%s retained label control characters", format)
		}
		if !strings.Contains(output.String(), `safe\n\x1b[31munsafe`) {
			t.Fatalf("%s does not contain readable escaped label: %q", format, output.String())
		}
	}
}
