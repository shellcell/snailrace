package report

import (
	"strings"
	"testing"
)

func TestWrapTextSplitsLongUnbrokenArguments(t *testing.T) {
	lines := wrapText("command "+strings.Repeat("x", 25), 10)
	for _, line := range lines {
		if len([]rune(line)) > 10 {
			t.Fatalf("line %q exceeds width", line)
		}
	}
}
