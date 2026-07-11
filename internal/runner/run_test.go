package runner

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunOnceUsesInjectedOutputWriter(t *testing.T) {
	var output bytes.Buffer
	_, err := runOnce(
		context.Background(),
		Spec{Name: "output", Shell: "printf visible; printf error >&2"},
		time.Millisecond,
		Options{ShowOutput: true, Output: &output},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"visible", "error"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("injected output does not contain %q: %q", expected, output.String())
		}
	}
}
