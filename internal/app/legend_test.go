package app

import (
	"bytes"
	"strings"
	"testing"

	"perftool/internal/runner"
)

func TestPrintCommandLegendPlainForNonTerminal(t *testing.T) {
	var buffer bytes.Buffer
	printCommandLegend(&buffer, []runner.Spec{
		{Name: "grep", Shell: "grep needle data.txt"},
		{Name: "sleep", Args: []string{"sleep", "0.1"}},
	})
	text := buffer.String()
	if strings.Contains(text, "\x1b[") {
		t.Fatalf("non-terminal legend should have no ANSI escapes: %q", text)
	}
	for _, want := range []string{"1. grep   grep needle data.txt", "2. sleep  sleep 0.1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("legend missing %q:\n%s", want, text)
		}
	}
}
