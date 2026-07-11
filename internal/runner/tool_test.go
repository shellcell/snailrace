package runner

import (
	"path/filepath"
	"testing"
)

func TestShellExecutableFindsSimpleCommand(t *testing.T) {
	got := shellExecutable("sh -c true")
	if filepath.Base(got) != "sh" {
		t.Fatalf("executable = %q, want sh", got)
	}
}

func TestShellExecutableRejectsCompoundCommand(t *testing.T) {
	if got := shellExecutable("'quoted tool' --flag"); got != "" {
		t.Fatalf("executable = %q, want empty", got)
	}
}
