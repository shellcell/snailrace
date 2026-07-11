//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCountDirectoryDoesNotRequireSortedEntries(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"z", "a", "middle"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := countDirectory(directory); got != 3 {
		t.Fatalf("directory count = %d, want 3", got)
	}
}

func TestMissingRootIsNotAZeroSample(t *testing.T) {
	if _, valid := SampleTree(1 << 30); valid {
		t.Fatal("missing root process should produce an invalid sample")
	}
}
