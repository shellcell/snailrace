package runner

import (
	"fmt"
	"os"
)

const pinExecutionSupported = true

// The child resolves the parent's /proc entry, which stays alive as long as
// the parent keeps the descriptor open — no fd inheritance is needed, so
// pinnedExtraFiles has nothing to pass down.
func pinnedExecutablePath(file *os.File) string {
	return fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), file.Fd())
}

func pinnedExtraFiles(*os.File) []*os.File { return nil }
