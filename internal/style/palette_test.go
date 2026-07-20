package style

import "testing"

func TestToolColorHandlesNegativeIndex(t *testing.T) {
	if got, want := Tool(-1), Tool(len(toolPalette)-1); got != want {
		t.Fatalf("negative color = %+v, want %+v", got, want)
	}
}
