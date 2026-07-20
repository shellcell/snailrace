package style

import "testing"

func TestCommandTextQuotesArgumentsWithoutChangingShellCommands(t *testing.T) {
	if got, want := CommandText(
		[]string{"tool", "two words", "", "it's", "line\n"}, false,
	), `tool 'two words' '' 'it'"'"'s' "line\n"`; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
	if got := CommandText([]string{"printf '%s' value"}, true); got != "printf '%s' value" {
		t.Fatalf("shell command = %q", got)
	}
}
